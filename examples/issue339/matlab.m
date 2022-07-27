function [net,stats] = cnn_train_dag(net, imdb, getBatch, varargin)
%CNN_TRAIN_DAG Demonstrates training a CNN using the DagNN wrapper
%    CNN_TRAIN_DAG() is similar to CNN_TRAIN(), but works with
%    the DagNN wrapper instead of the SimpleNN wrapper.

% Copyright (C) 2014-16 Andrea Vedaldi.
% All rights reserved.
%
% This file is part of the VLFeat library and is made available under
% the terms of the BSD license (see the COPYING file).

opts.expDir = fullfile('data','exp') ;
opts.continue = true ;
opts.batchSize = 256 ;
opts.numSubBatches = 1 ;
opts.averageImage = [];
opts.train = [] ;
opts.val = [] ;
opts.gpus = [] ;
opts.prefetch = false ;
opts.numEpochs = 300 ;
opts.gamma = 0.9;
opts.constraint = 10;
opts.learningRate = 0.001 ;
opts.weightDecay = 0.0005 ;
opts.momentum = 0.9 ;
opts.saveMomentum = true ;
opts.nesterovUpdate = false ;
opts.randomSeed = 0 ;
opts.profile = false ;
opts.parameterServer.method = 'mmap' ;
opts.parameterServer.prefix = 'mcn' ;

opts.derOutputs = {'objective', 1} ;
opts.extractStatsFn = @extractStats ;
opts.plotStatistics = true;
opts = vl_argparse(opts, varargin) ;

if ~exist(opts.expDir, 'dir'), mkdir(opts.expDir) ; end
if isempty(opts.train), opts.train = find(imdb.images.set==1) ; end
if isempty(opts.val), opts.val = find(imdb.images.set==2) ; end
if isnan(opts.train), opts.train = [] ; end
if isnan(opts.val), opts.val = [] ; end

% -------------------------------------------------------------------------
%                                                            Initialization
% -------------------------------------------------------------------------

evaluateMode = isempty(opts.train) ;
if ~evaluateMode
    if isempty(opts.derOutputs)
        error('DEROUTPUTS must be specified when training.\n') ;
    end
end

% -------------------------------------------------------------------------
%                                                        Train and validate
% -------------------------------------------------------------------------

modelPath = @(ep) fullfile(opts.expDir, sprintf('net-epoch-%d.mat', ep));
modelFigPath = fullfile(opts.expDir, 'net-train.pdf') ;

start = opts.continue * findLastCheckpoint(opts.expDir) ;
if start >= 1
    fprintf('%s: resuming by loading epoch %d\n', mfilename, start) ;
    [net, state, stats] = loadState(modelPath(start)) ;
else
    state = [] ;
end

for epoch=start+1:opts.numEpochs
    
    % Set the random seed based on the epoch and opts.randomSeed.
    % This is important for reproducibility, including when training
    % is restarted from a checkpoint.
    
    rng(epoch + opts.randomSeed) ;
    prepareGPUs(opts, epoch == start+1) ;
    
    % Train for one epoch.
    params = opts ;
    params.epoch = epoch ;
    params.learningRate = opts.learningRate(min(epoch, numel(opts.learningRate))) ;
    params.train = opts.train(randperm(numel(opts.train))) ; % shuffle
    params.val = opts.val(randperm(numel(opts.val))) ;
    params.imdb = imdb ;
    params.getBatch = getBatch ;
    
    if numel(opts.gpus) <= 1
        [net, state] = processEpoch(net, state, params, 'train',opts) ;
        [net, state] = processEpoch(net, state, params, 'val',opts) ;
        if ~evaluateMode
            saveState(modelPath(epoch), net, state) ;
        end
        lastStats = state.stats ;
    else
        spmd
            [net, state] = processEpoch(net, state, params, 'train',opts) ;
            [net, state] = processEpoch(net, state, params, 'val',opts) ;
            if labindex == 1 && ~evaluateMode
                saveState(modelPath(epoch), net, state) ;
            end
            lastStats = state.stats ;
        end
        lastStats = accumulateStats(lastStats) ;
    end
    
    stats.train(epoch) = lastStats.train ;
    stats.val(epoch) = lastStats.val ;
    clear lastStats ;
    saveStats(modelPath(epoch), stats) ;
    
    if opts.plotStatistics
        switchFigure(1) ; clf ;
        plots = setdiff(...
            cat(2,...
            fieldnames(stats.train)', ...
            fieldnames(stats.val)'), {'num', 'time'}) ;
        for p = plots
            p = char(p) ;
            values = zeros(0, epoch) ;
            leg = {} ;
            for f = {'train', 'val'}
                f = char(f) ;
                if isfield(stats.(f), p)
                    tmp = [stats.(f).(p)] ;
                    values(end+1,:) = tmp(1,:)' ;
                    leg{end+1} = f ;
                end
            end
            subplot(1,numel(plots),find(strcmp(p,plots))) ;
            plot(1:epoch, values','o-') ;
            xlabel('epoch') ;
            title(p) ;
            legend(leg{:}) ;
            grid on ;
        end
        drawnow ;
        print(1, modelFigPath, '-dpdf') ;
    end
end

% With multiple GPUs, return one copy
if isa(net, 'Composite'), net = net{1} ; end

% -------------------------------------------------------------------------
function [net, state] = processEpoch(net, state, params, mode,opts)
% -------------------------------------------------------------------------
% Note that net is not strictly needed as an output argument as net
% is a handle class. However, this fixes some aliasing issue in the
% spmd caller.

% initialize with momentum 0
if isempty(state) || isempty(state.momentum)
    state.r = num2cell(zeros(1, numel(net.params))) ;
    state.momentum = num2cell(zeros(1, numel(net.params))) ;
end

% move CNN  to GPU as needed
numGpus = numel(params.gpus) ;
if numGpus >= 1
    net.move('gpu') ;
    state.momentum = cellfun(@gpuArray, state.momentum, 'uniformoutput', false) ;
    %state.r = cellfun(@gpuArray, state.r, 'uniformoutput', false) ;
end
if numGpus > 1
    parserv = ParameterServer(params.parameterServer) ;
    net.setParameterServer(parserv) ;
else
    parserv = [] ;
end

% profile
if params.profile
    if numGpus <= 1
        profile clear ;
        profile on ;
    else
        mpiprofile reset ;
        mpiprofile on ;
    end
end

num = 0 ;
epoch = params.epoch ;
subset = params.(mode) ;
adjustTime = 0 ;

stats.num = 0 ; % return something even if subset = []
stats.time = 0 ;

start = tic ;
for t=1:params.batchSize:numel(subset)
    fprintf('%s: epoch %02d: %3d/%3d:', mode, epoch, ...
        fix((t-1)/params.batchSize)+1, ceil(numel(subset)/params.batchSize)) ;
    batchSize = min(params.batchSize, numel(subset) - t + 1) ;
    
    for s=1:params.numSubBatches
        % get this image batch and prefetch the next
        batchStart = t + (labindex-1) + (s-1) * numlabs ;
        batchEnd = min(t+params.batchSize-1, numel(subset)) ;
        batch = subset(batchStart : params.numSubBatches * numlabs : batchEnd) ;
        num = num + numel(batch) ;
        if numel(batch) == 0, continue ; end
        opts.epoch = params.epoch;
        inputs = params.getBatch(params.imdb, batch,opts) ;
        %------------zzd
        %if(opts.epoch>10)
         %   net.params(net.getParamIndex('lof')).learningRate = 1e-4;
         %   net.params(net.getParamIndex('lob')).learningRate = 2e-4;
        %end
        
        if params.prefetch
            if s == params.numSubBatches
                batchStart = t + (labindex-1) + params.batchSize ;
                batchEnd = min(t+2*params.batchSize-1, numel(subset)) ;
            else
                batchStart = batchStart + numlabs ;
            end
            nextBatch = subset(batchStart : params.numSubBatches * numlabs : batchEnd) ;
            params.getBatch(params.imdb, nextBatch,opts) ;
        end
        
        if strcmp(mode, 'train')
            net.mode = 'normal' ;
            net.accumulateParamDers = (s ~= 1) ;
            net.eval(inputs, params.derOutputs, 'holdOn', s < params.numSubBatches) ;
        else
            net.mode = 'test' ;
            net.eval(inputs) ;
        end
    end
    
    % Accumulate gradient.
    if strcmp(mode, 'train')
        if ~isempty(parserv), parserv.sync() ; end
        state = accumulateGradients(net, state, params, batchSize, parserv) ;
    end
    
    % Get statistics.
    time = toc(start) + adjustTime ;
    batchTime = time - stats.time ;
    stats.num = num ;
    stats.time = time ;
    stats = params.extractStatsFn(stats,net) ;
    currentSpeed = batchSize / batchTime ;
    averageSpeed = (t + batchSize - 1) / time ;
    if t == 3*params.batchSize + 1
        % compensate for the first three iterations, which are outliers
        adjustTime = 4*batchTime - time ;
        stats.time = time + adjustTime ;
    end
    
    fprintf(' %.1f (%.1f) Hz', averageSpeed, currentSpeed) ;
    for f = setdiff(fieldnames(stats)', {'num', 'time'})
        f = char(f) ;
        fprintf(' %s: %.3f', f, stats.(f)) ;
    end
    fprintf('\n') ;
end

% Save back to state.
state.stats.(mode) = stats ;
if params.profile
    if numGpus <= 1
        state.prof.(mode) = profile('info') ;
        profile off ;
    else
        state.prof.(mode) = mpiprofile('info');
        mpiprofile off ;
    end
end
if ~params.saveMomentum
    state.momentum = [] ;
else
    state.momentum = cellfun(@gather, state.momentum, 'uniformoutput', false) ;
end

net.reset() ;
net.move('cpu') ;

% -------------------------------------------------------------------------
function state = accumulateGradients(net, state, params, batchSize, parserv)
% -------------------------------------------------------------------------
numGpus = numel(params.gpus) ;
otherGpus = setdiff(1:numGpus, labindex) ;

for p=1:numel(net.params)
    
    if ~isempty(parserv)
        parDer = parserv.pullWithIndex(p) ;
    else
        parDer = net.params(p).der ;
        %parDer(parDer>params.constraint) = params.constraint;
        %parDer(parDer<-params.constraint) = -params.constraint;
    end
    
    switch net.params(p).trainMethod
        
        case 'average' % mainly for batch normalization
            thisLR = net.params(p).learningRate ;
            net.params(p).value = vl_taccum(...
                1 - thisLR, net.params(p).value, ...
                (thisLR/batchSize/net.params(p).fanout),  parDer) ;
            
        case 'gradient'
            thisDecay = params.weightDecay * net.params(p).weightDecay ;
            thisLR = params.learningRate * net.params(p).learningRate ;
            
            if thisLR>0 || thisDecay>0
                % Normalize gradient and incorporate weight decay.
                parDer = vl_taccum(1/batchSize, parDer, ...
                    thisDecay, net.params(p).value) ;
                
                % Update momentum.
                state.momentum{p} = vl_taccum(...
                    params.momentum, state.momentum{p}, ...
                    -1, parDer) ;
                
                % Nesterov update (aka one step ahead).
                if params.nesterovUpdate
                    delta = vl_taccum(...
                        params.momentum, state.momentum{p}, ...
                        -1, parDer) ;
                else
                    delta = state.momentum{p} ;
                end
                
                % Update parameters.
                net.params(p).value = vl_taccum(...
                    1,  net.params(p).value, thisLR, delta) ;
            end
        case 'rmsprop'
            opts = params;
            thisDecay = opts.weightDecay * net.params(p).weightDecay ;
            thisLR = opts.learningRate * net.params(p).learningRate ;
            state.r{p} = (1 - opts.gamma) * (net.params(p).der).^2 + opts.gamma * state.r{p};
            der_constraint = (1 / batchSize) * net.params(p).der./ (sqrt(state.r{p})+1e-8);
            der_constraint(der_constraint>opts.constraint) = opts.constraint;
            der_constraint(der_constraint<-opts.constraint) = -opts.constraint;
            state.momentum{p} = opts.momentum * state.momentum{p} ...
                - thisDecay * net.params(p).value ...
                - der_constraint ;
            net.params(p).value = net.params(p).value  + thisLR * state.momentum{p} ;
        otherwise
            error('Unknown training method ''%s'' for parameter ''%s''.', ...
                net.params(p).trainMethod, ...
                net.params(p).name) ;
    end
end

% -------------------------------------------------------------------------
function stats = accumulateStats(stats_)
% -------------------------------------------------------------------------

for s = {'train', 'val'}
    s = char(s) ;
    total = 0 ;
    
    % initialize stats stucture with same fields and same order as
    % stats_{1}
    stats__ = stats_{1} ;
    names = fieldnames(stats__.(s))' ;
    values = zeros(1, numel(names)) ;
    fields = cat(1, names, num2cell(values)) ;
    stats.(s) = struct(fields{:}) ;
    
    for g = 1:numel(stats_)
        stats__ = stats_{g} ;
        num__ = stats__.(s).num ;
        total = total + num__ ;
        
        for f = setdiff(fieldnames(stats__.(s))', 'num')
            f = char(f) ;
            stats.(s).(f) = stats.(s).(f) + stats__.(s).(f) * num__ ;
            
            if g == numel(stats_)
                stats.(s).(f) = stats.(s).(f) / total ;
            end
        end
    end
    stats.(s).num = total ;
end

% -------------------------------------------------------------------------
function stats = extractStats(stats, net)
% -------------------------------------------------------------------------
sel = find(cellfun(@(x) isa(x,'dagnn.Loss'), {net.layers.block})) ;
for i = 1:numel(sel)
    stats.(net.layers(sel(i)).outputs{1}) = net.layers(sel(i)).block.average ;
end

% -------------------------------------------------------------------------
function saveState(fileName, net_, state)
% -------------------------------------------------------------------------
net = net_.saveobj() ;
state.r = [];
save(fileName, 'net', 'state') ;

% -------------------------------------------------------------------------
function saveStats(fileName, stats)
% -------------------------------------------------------------------------
if exist(fileName)
    save(fileName, 'stats', '-append') ;
else
    save(fileName, 'stats') ;
end

% -------------------------------------------------------------------------
function [net, state, stats] = loadState(fileName)
% -------------------------------------------------------------------------
load(fileName, 'net', 'state', 'stats') ;
net = dagnn.DagNN.loadobj(net) ;
if isempty(whos('stats'))
    error('Epoch ''%s'' was only partially saved. Delete this file and try again.', ...
        fileName) ;
end

% -------------------------------------------------------------------------
function epoch = findLastCheckpoint(modelDir)
% -------------------------------------------------------------------------
list = dir(fullfile(modelDir, 'net-epoch-*.mat')) ;
tokens = regexp({list.name}, 'net-epoch-([\d]+).mat', 'tokens') ;
epoch = cellfun(@(x) sscanf(x{1}{1}, '%d'), tokens) ;
epoch = max([epoch 0]) ;

% -------------------------------------------------------------------------
function switchFigure(n)
% -------------------------------------------------------------------------
if get(0,'CurrentFigure') ~= n
    try
        set(0,'CurrentFigure',n) ;
    catch
        figure(n) ;
    end
end

% -------------------------------------------------------------------------
function clearMex()
% -------------------------------------------------------------------------
clear vl_tmove vl_imreadjpeg ;

% -------------------------------------------------------------------------
function prepareGPUs(opts, cold)
% -------------------------------------------------------------------------
numGpus = numel(opts.gpus) ;
if numGpus > 1
    % check parallel pool integrity as it could have timed out
    pool = gcp('nocreate') ;
    if ~isempty(pool) && pool.NumWorkers ~= numGpus
        delete(pool) ;
    end
    pool = gcp('nocreate') ;
    if isempty(pool)
        parpool('local', numGpus) ;
        cold = true ;
    end
    
end
if numGpus >= 1 && cold
    fprintf('%s: resetting GPU\n', mfilename)
    clearMex() ;
    if numGpus == 1
        gpuDevice(opts.gpus)
    else
        spmd
            clearMex() ;
            gpuDevice(opts.gpus(labindex))
        end
    end
end

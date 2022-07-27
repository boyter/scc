//
//  SGRtmpSession.m
//  SGLivingPublisher
//
//  Created by iossinger on 16/6/16.
//  Copyright © 2016年 iossinger. All rights reserved.
//

#import "SGRtmpSession.h"
#import "SGStreamSession.h"
#import "SGRtmpTypes.h"
#import "NSString+URL.h"
#import "NSMutableData+Buffer.h"
#import "SGRtmpConfig.h"

static const size_t kRTMPSignatureSize = 1536;


@interface SGRtmpSession()<SGStreamSessionDelegate>
{
    //两个线程,一个负责组装数据,一个负责发送数据
    dispatch_queue_t _packetQueue;
    dispatch_queue_t _sendQueue;
    
    int _outChunkSize;
    int _inChunkSize;
    int _streamID;
    int _numOfInvokes;
}

@property (nonatomic,strong) SGStreamSession *session;

@property (nonatomic,  copy) NSString *url;

@property (nonatomic,assign) SGRtmpSessionStatus rtmpStatus;

@property (nonatomic,strong) NSMutableData *handshake;

@property (nonatomic,strong) NSMutableDictionary *preChunk;

@property (nonatomic,strong) NSMutableDictionary<NSNumber *,NSString *> *trackedCommands;

@end


@implementation SGRtmpSession

- (void)dealloc{
    NSLog(@"%s",__func__);
    [self sendDeleteStream];
    self.url = nil;
    self.delegate = nil;
    self.session.delegate = nil;
    self.session = nil;
    _packetQueue = nil;
    _sendQueue = nil;
    _rtmpStatus = SGRtmpSessionStatusNone;
    _numOfInvokes = 0;
    [_preChunk removeAllObjects];
    [_trackedCommands removeAllObjects];
    _config = nil;
}

- (NSMutableDictionary<NSNumber *,NSString *> *)trackedCommands{
    if (_trackedCommands == nil) {
        _trackedCommands = [NSMutableDictionary dictionary];
    }
    return _trackedCommands;
}

- (NSMutableDictionary *)preChunk{
    if (_preChunk == nil) {
        _preChunk = [NSMutableDictionary dictionary];
    }
    return _preChunk;
}

- (SGStreamSession *)session{
    if (_session == nil) {
        _session = [[SGStreamSession alloc] init];
        _session.delegate = self;
    }
    return _session;
}

- (void)setUrl:(NSString *)url{
    _url = url;
    NSLog(@"scheme:%@",url.scheme);
    NSLog(@"host:%@",url.host);
    NSLog(@"app:%@",url.app);
    NSLog(@"playPath:%@",url.playPath);
    NSLog(@"port:%zd",url.port);
}

- (void)setRtmpStatus:(SGRtmpSessionStatus)rtmpStatus{
    _rtmpStatus = rtmpStatus;
    NSLog(@"rtmpStatus-----%zd",rtmpStatus);
    if ([self.delegate respondsToSelector:@selector(rtmpSession:didChangeStatus:)]) {
        [self.delegate rtmpSession:self didChangeStatus:_rtmpStatus];
    }
}

- (instancetype)init{
   
    if (self = [super init]) {
        
        _rtmpStatus = SGRtmpSessionStatusNone;
        _packetQueue = dispatch_queue_create("packet", 0);
        _sendQueue = dispatch_queue_create("send", 0);
        
        _outChunkSize = 128;
        _inChunkSize  = 128;
    }
    
    return self;
    
}

- (void)setConfig:(SGRtmpConfig *)config{
    _config = config;
    self.url = config.url;
}
- (void)connect{
    [self.session connectToServer:self.url.host port:self.url.port];
}
- (void)disConnect{
    [self reset];
    [self.session disConnect];
}
- (void)reset{
    self.handshake = nil;
    self.preChunk  = nil;
    self.trackedCommands = nil;
    _streamID = 0;
    _numOfInvokes = 0;
    _outChunkSize = 128;
    _inChunkSize  = 128;
    self.rtmpStatus = SGRtmpSessionStatusNone;
}
#pragma mark -------delegate---------
- (void)streamSession:(SGStreamSession *)session didChangeStatus:(SGStreamStatus)streamStatus{
    
    if (streamStatus & NSStreamEventHasBytesAvailable) {//收到数据
        [self didReceivedata];
        return;//return
    }
    
    if (streamStatus & NSStreamEventHasSpaceAvailable){ //可以写数据
        
        if (_rtmpStatus == SGRtmpSessionStatusConnected) {
           [self handshake0];
        }
        
        return;//return
    }
    
    if ((streamStatus & NSStreamEventOpenCompleted) &&
        _rtmpStatus < SGRtmpSessionStatusConnected) {
        self.rtmpStatus = SGRtmpSessionStatusConnected;
    }
    
    if (streamStatus & NSStreamEventErrorOccurred) {
        self.rtmpStatus = SGRtmpSessionStatusError;
    }
    
    if (streamStatus & NSStreamEventEndEncountered) {
        self.rtmpStatus = SGRtmpSessionStatusNotConnected;
    }
}

- (void)handshake0{
    
    self.rtmpStatus = SGRtmpSessionStatusHandshake0;
    
    //c0
    char c0Byte = 0x03;
    NSData *c0 = [NSData dataWithBytes:&c0Byte length:1];
    [self writeData:c0];
    
    //c1
    uint8_t *c1Bytes = (uint8_t *)malloc(kRTMPSignatureSize);
    memset(c1Bytes, 0, 4 + 4);
    NSData *c1 = [NSData dataWithBytes:c1Bytes length:kRTMPSignatureSize];
    free(c1Bytes);
    [self writeData:c1];
}

- (void)handshake1{
    self.rtmpStatus = SGRtmpSessionStatusHandshake2;
    NSData *s1 = [self.handshake subdataWithRange:NSMakeRange(0, kRTMPSignatureSize)];
    //c2
    uint8_t *s1Bytes = (uint8_t *)s1.bytes;
    memset(s1Bytes + 4, 0, 4);
    NSData *c2 = [NSData dataWithBytes:s1Bytes length:s1.length];
    [self writeData:c2];
}

//验证过
- (void)sendConnectPacket{
    NSLog(@"sendConnectPacket");
//    AMF格式
    RTMPChunk_0 metadata = {0};
    metadata.msg_stream_id = SGStreamIDInvoke;
    metadata.msg_type_id = SGMSGTypeID_INVOKE;
    
    NSString *url;
    NSMutableData *buff = [NSMutableData data];
    if (_url.port > 0) {
        url = [NSString stringWithFormat:@"%@://%@:%zd/%@",_url.scheme,_url.host,_url.port,_url.app];
    }else{
        url = [NSString stringWithFormat:@"%@://%@/%@",_url.scheme,_url.host,_url.app];
    }
    
    [buff appendString:@"connect"];
    [buff appendDouble:++_numOfInvokes];
    self.trackedCommands[@(_numOfInvokes)] = @"connect";
    [buff appendByte:kAMFObject];
    [buff putKey:@"app" stringValue:_url.app];
    [buff putKey:@"type" stringValue:@"nonprivate"];
    [buff putKey:@"tcUrl" stringValue:url];
    [buff putKey:@"fpad" boolValue:NO];//是否使用代理
    [buff putKey:@"capabilities" doubleValue:15.];
    [buff putKey:@"audioCodecs" doubleValue:10.];
    [buff putKey:@"videoCodecs" doubleValue:7.];
    [buff putKey:@"videoFunction" doubleValue:1.];
    [buff appendByte16:0];
    [buff appendByte:kAMFObjectEnd];
    
    metadata.msg_length.data = (int)buff.length;
    [self sendPacket:buff :metadata];
}

- (void)sendPacket:(NSData *)data :(RTMPChunk_0)metadata{
    
    SGFrame *frame = [[SGFrame alloc] init];
    
    frame.data = data;
    frame.timestamp = metadata.timestamp.data;
    frame.msgLength = metadata.msg_length.data;
    frame.msgTypeId = metadata.msg_type_id;//消息类型
    frame.msgStreamId = metadata.msg_stream_id;//消息流id
    
    [self sendBuffer:frame];
}
/**
 *  Chunk Basic Header: HeaderType+ChannelID组成  1个字节
 *     >HeaderType(前两bit): 00->12字节  01->8字节
 *     >ChannelID(后6个bit): 02->Ping和ByteRead通道 03->Invoke通道 connect() publish()和自己写的NetConnection.Call() 04->Audio和Vidio通道
 *
 *  12字节举例
 *  Chunk Message Header:timestamp + message_length+message_typ + msg_stream_id
 *  message_typ :type为1,2,3,5,6的时候是协议控制消息
 *
 *               type为4的时候表示 User Control Messages [Event_type + Event_Data] Event_type有Stream Begin，Stream End...
 *
 *               type为8，音频数据
 *
 *               type为9，视频数据
 *
 *               type为18 元数据消息[AMF0]
 *
 *               type为20 命令消息 Command Message(RPC Message)
 *               例如connect, createStream, publish, play, pause on the peer
 *
 *
 *
 */
- (void)sendBuffer:(SGFrame *)frame{
   dispatch_sync(_packetQueue, ^{
    
       uint64_t ts = frame.timestamp;
       
       int streamId = frame.msgStreamId;
       NSLog(@"streamId------%d",streamId);
       NSNumber *preTimestamp = self.preChunk[@(streamId)];
       
       uint8_t *chunk;
       int offset = 0;
       
       if (preTimestamp == nil) {//第一帧,音频或者视频
           chunk = malloc(12);
           chunk[0] = RTMP_CHUNK_TYPE_0/*0x00*/ | (streamId & 0x1F); //前两个字节 00 表示12字节
           offset += 1;
           
           memcpy(chunk+offset, [NSMutableData be24:(uint32_t)ts], 3);
           offset += 3;//时间戳3个字节
           
           memcpy(chunk+offset, [NSMutableData be24:frame.msgLength], 3);
           offset += 3;//消息长度3个字节
           
           int msgTypeId = frame.msgTypeId;//一个字节的消息类型
           memcpy(chunk+offset, &msgTypeId, 1);
           offset += 1;
           
           memcpy(chunk+offset, (uint8_t *)&(_streamID), sizeof(_streamID));
           offset += sizeof(_streamID);
           
       }else{//不是第一帧
           chunk = malloc(8);
           chunk[0] = RTMP_CHUNK_TYPE_1/*0x40*/ | (streamId & 0x1F);//前两个字节01表示8字节
           offset += 1;
           
           char *temp = [NSMutableData be24:(uint32_t)(ts - preTimestamp.integerValue)];
           memcpy(chunk+offset, temp, 3);
           offset += 3;
           
           memcpy(chunk+offset, [NSMutableData be24:frame.msgLength], 3);
           offset += 3;
           
           int msgTypeId = frame.msgTypeId;
           memcpy(chunk+offset, &msgTypeId, 1);
           offset += 1;
       }

       self.preChunk[@(streamId)] = @(ts);
       
       uint8_t *bufferData = (uint8_t *)frame.data.bytes;
       uint8_t *outp = (uint8_t *)malloc(frame.data.length + 64);
       memcpy(outp, chunk, offset);
       free(chunk);
       
       NSUInteger total = frame.data.length;
       NSInteger step = MIN(total, _outChunkSize);
       
       memcpy(outp+offset, bufferData, step);
       offset += step;
       total  -= step;
       bufferData += step;
       
       while (total > 0) {
           step = MIN(total, _outChunkSize);
           bufferData[-1] = RTMP_CHUNK_TYPE_3/*0xC0*/ | (streamId & 0x1F);//11表示一个字节,直接跳过这个字节;
           memcpy(outp+offset, bufferData - 1, step + 1);
           
           offset += step + 1;
           total  -= step;
           bufferData += step;
       }
       
       NSData *tosend = [NSData dataWithBytes:outp length:offset];
       free(outp);
       [self writeData:tosend];
   });
}

//接收到数据
- (void)didReceivedata{
    NSData *data = [self.session readData];
    
    if (self.rtmpStatus >= SGRtmpSessionStatusConnected &&
        self.rtmpStatus < SGRtmpSessionStatusHandshakeComplete) {
        [self.handshake appendData:data];
    }
    
    NSLog(@"%zd",data.length);
    
//handshke 可能情况: 1.按照官方文档c0,c1,c2
    //          2.一起发3073个字节
    //          3.先发一部分,再发一部分,每部分大小不确定,总数3073正确
    switch (_rtmpStatus) {
        case SGRtmpSessionStatusHandshake0:{
            uint8_t s0;
            [data getBytes:&s0 length:1];
            if (s0 == 0x03) {//s0
                self.rtmpStatus = SGRtmpSessionStatusHandshake1;
                if (data.length > 1) {//后面还有数据,但不确定长度
                    data = [data subdataWithRange:NSMakeRange(1, data.length -1)];
                    self.handshake = data.mutableCopy;
                }else{
                    break;
                }
            }else{
                NSLog(@"握手失败");
                break;
            }
        }
        case SGRtmpSessionStatusHandshake1:{
            
            if (self.handshake.length >= kRTMPSignatureSize) {//s1
                [self handshake1];
                
                if (self.handshake.length > kRTMPSignatureSize) {//>
                    NSData *subData = [self.handshake subdataWithRange:NSMakeRange(kRTMPSignatureSize, self.handshake.length - kRTMPSignatureSize)];
                    self.handshake = subData.mutableCopy;
                }else{// =
                    self.handshake = [NSMutableData data];
                    break;
                }
            }else{// <
                break;
            }
        }
            
        case SGRtmpSessionStatusHandshake2:{//s2
            if (data.length >= kRTMPSignatureSize) {
                NSLog(@"握手完成");
                self.rtmpStatus = SGRtmpSessionStatusHandshakeComplete;
                [self sendConnectPacket];
            }
            break;
        }
        default:
            [self parseData:data];
            break;
    }
}

- (void)parseData:(NSData *)data{
  
    if (data.length == 0) {
        return;
    }
    
    uint8_t *buffer = (uint8_t *)data.bytes;
    NSUInteger total = data.length;
    
    while (total > 0) {
        int headType = (buffer[0] & 0xC0) >> 6;//取出前两个字节
        buffer++;
        total --;
        
        if (total <= 0) {
            break;
        }
        
        switch (headType) {
            case RTMP_HEADER_TYPE_FULL:{
                RTMPChunk_0 chunk;
                memcpy(&chunk, buffer, sizeof(RTMPChunk_0));
                chunk.msg_length.data = [NSMutableData getByte24:(uint8_t *)&chunk.msg_length];
                buffer += sizeof(RTMPChunk_0);
                total  -= sizeof(RTMPChunk_0);
                BOOL isSuccess = [self handleMeesage:buffer :chunk.msg_type_id];
                if (!isSuccess) {
                    total = 0;break;
                }
            
                buffer += chunk.msg_length.data;
                total  -= chunk.msg_length.data;
            }
                break;
            case RTMP_HEADER_TYPE_NO_MSG_STREAM_ID:{
                RTMPChunk_1 chunk;
                memcpy(&chunk, buffer, sizeof(RTMPChunk_1));
                buffer += sizeof(RTMPChunk_1);
                total  -= sizeof(RTMPChunk_1);
                chunk.msg_length.data = [NSMutableData getByte24:(uint8_t *)&chunk.msg_length];
                BOOL isSuccess = [self handleMeesage:buffer :chunk.msg_type_id];
                if (!isSuccess) {
                    total = 0;break;
                }
                
                buffer += chunk.msg_length.data;
                total  -= chunk.msg_length.data;
            }
                break;
            case RTMP_HEADER_TYPE_TIMESTAMP:{
                RTMPChunk_2 chunk;
                memcpy(&chunk, buffer, sizeof(RTMPChunk_2));
                buffer += sizeof(RTMPChunk_2) + MIN(total, _inChunkSize);
                total  -= sizeof(RTMPChunk_2) + MIN(total, _inChunkSize);
                
            }
                break;
            case RTMP_HEADER_TYPE_ONLY:{
                buffer += MIN(total, _inChunkSize);
                total  -= MIN(total, _inChunkSize);
            }
                break;
                
            default:
                return;
        }
    }
}

- (BOOL)handleMeesage:(uint8_t *)p :(uint8_t)msgTypeId{
    BOOL handleSuccess = YES;
    switch(msgTypeId) {
        case SGMSGTypeID_BYTES_READ:
        {
            
        }
            break;
            
        case SGMSGTypeID_CHUNK_SIZE:
        {
            unsigned long newChunkSize = [NSMutableData getByte32:p];//get_be32(p);
            NSLog(@"change incoming chunk size from %d to: %zu", _inChunkSize, newChunkSize);
            _inChunkSize = (int)newChunkSize;
        }
            break;
            
        case SGMSGTypeID_PING:
        {
            NSLog(@"received ping, sending pong.");
            [self sendPong];
        }
            break;
            
        case SGMSGTypeID_SERVER_WINDOW:
        {
            NSLog(@"received server window size: %d\n", [NSMutableData getByte32:p]);
        }
            break;
            
        case SGMSGTypeID_PEER_BW:
        {
            NSLog(@"received peer bandwidth limit: %d type: %d\n", [NSMutableData getByte32:p], p[4]);
        }
            break;
            
        case SGMSGTypeID_INVOKE:
        {
            NSLog(@"Received invoke");
            [self handleInvoke:p];//handleInvoke
        }
            break;
        case SGMSGTypeID_VIDEO:
        {
            NSLog(@"received video");
        }
            break;
            
        case SGMSGTypeID_AUDIO:
        {
            NSLog(@"received audio");
        }
            break;
            
        case SGMSGTypeID_METADATA:
        {
            NSLog(@"received metadata");
        }
            break;
            
        case SGMSGTypeID_NOTIFY:
        {
            NSLog(@"received notify");
        }
            break;
            
        default:
        {
            NSLog(@"received unknown packet type: 0x%02X", msgTypeId);
            handleSuccess = NO;
        }
            break;
    }
    return handleSuccess;
}

- (void)sendPong{
    dispatch_sync(_packetQueue, ^{
        int streamId = 0;
        
        NSMutableData *data = [NSMutableData data];
        [data appendByte:2];
        [data appendByte24:0];
        [data appendByte24:6];
        [data appendByte:SGMSGTypeID_PING];
        
        [data appendBytes:(uint8_t*)&streamId length:sizeof(int32_t)];
        [data appendByte16:7];
        [data appendByte16:0];
        [data appendByte16:0];

        [self writeData:data];
    });
}

- (void)handleInvoke:(uint8_t *)p{
    int buflen = 0;
    NSString *command = [NSMutableData getString:p :&buflen];
    NSLog(@"received invoke %@\n", command);
    
    int pktId = (int)[NSMutableData getDouble:p + 11];
    NSLog(@"pktId: %d\n", pktId);

    NSString *trackedCommand = self.trackedCommands[@(pktId)] ;
    
    if ([command isEqualToString:@"_result"]) {
        NSLog(@"tracked command: %@\n", trackedCommand);
        if ([trackedCommand isEqualToString:@"connect"]) {
            [self sendReleaseStream];
            [self sendFCPublish];
            [self sendCreateStream];
            self.rtmpStatus = SGRtmpSessionStatusFCPublish;
        } else if ([trackedCommand isEqualToString:@"createStream"]) {
            if (p[10] || p[19] != 0x05 || p[20]) {
                NSLog(@"RTMP: Unexpected reply on connect()\n");
            } else {
                _streamID = [NSMutableData getDouble:p+21];
            }
            [self sendPublish];
            self.rtmpStatus = SGRtmpSessionStatusReady;
        }
    } else if ([command isEqualToString:@"onStatus"]) {//parseStatusCode
        NSString *code = [self parseStatusCode:p + 3 + command.length];
        NSLog(@"code : %@", code);
        if ([code isEqualToString:@"NetStream.Publish.Start"]) {
            
            // [self sendHeaderPacket];//貌似不发这一句,也可以
            
            //重新设定了chunksize大小
            [self sendSetChunkSize:getpagesize()];//16K
            
            //sendSetBufferTime(0);//设定时间
            self.rtmpStatus = SGRtmpSessionStatusSessionStarted;
        }
    }
}

- (NSString *)parseStatusCode:(uint8_t *)p{
    NSMutableDictionary *props = [NSMutableDictionary dictionary];
    
    // skip over the packet id
    p += sizeof(double) + 1;
    
    //keep reading until we find an AMF Object
    bool foundObject = false;
    while (!foundObject) {
        if (p[0] == AMF_DATA_TYPE_OBJECT) {
            p += 1;
            foundObject = true;
            continue;
        } else {
            p += [self amfPrimitiveObjectSize:p];
        }
    }
    
    // read the properties of the object
    uint16_t nameLen, valLen;
    char propName[128], propVal[128];
    do {
        nameLen = [NSMutableData getByte16:p];//get_be16(p);
        p += sizeof(nameLen);
        strncpy(propName, (char*)p, nameLen);
        propName[nameLen] = '\0';
        p += nameLen;
        NSString *key = [NSString stringWithUTF8String:propName];
        NSLog(@"key----%@",key);
        if (p[0] == AMF_DATA_TYPE_STRING) {
            valLen = [NSMutableData getByte16:p+1];//get_be16(p+1);
            p += sizeof(valLen) + 1;
            strncpy(propVal, (char*)p, valLen);
            propVal[valLen] = '\0';
            p += valLen;
            NSString *value = [NSString stringWithUTF8String:propVal];
            props[key] = value;
        } else {
            // treat non-string property values as empty
            p += [self amfPrimitiveObjectSize:p];
            props[key] = @"";
        }
    } while ([NSMutableData getByte24:p] != AMF_DATA_TYPE_OBJECT_END);
    
    //p = start;
    return props[@"code"] ;
}

- (int)amfPrimitiveObjectSize:(uint8_t *)p{//amf原始对象
    switch(p[0]) {
        case AMF_DATA_TYPE_NUMBER:       return 9;
        case AMF_DATA_TYPE_BOOL:         return 2;
        case AMF_DATA_TYPE_NULL:         return 1;
        case AMF_DATA_TYPE_STRING:       return 3 + [NSMutableData getByte16:p];
        case AMF_DATA_TYPE_LONG_STRING:  return 5 + [NSMutableData getByte32:p];
    }
    return -1; // not a primitive, likely an object
}

- (void)sendHeaderPacket{
    RTMPChunk_0 metadata = {0};
    NSMutableData *buffer =[NSMutableData data];
    [buffer appendString:@"@setDataFrame"];
    [buffer appendString:@"onMetaData"];
    [buffer appendByte:kAMFObject];
    
    [buffer putKey:@"width" doubleValue:self.config.width];
    [buffer putKey:@"height" doubleValue:self.config.height];
    [buffer putKey:@"displaywidth" doubleValue:self.config.width];
    [buffer putKey:@"displayheight" doubleValue:self.config.height];
    [buffer putKey:@"framewidth" doubleValue:self.config.width];
    [buffer putKey:@"frameheight" doubleValue:self.config.height];
    [buffer putKey:@"videodatarate" doubleValue:self.config.videoBitrate / 1024.];
    [buffer putKey:@"videoframerate" doubleValue:1.0 / self.config.frameDuration];
    
    [buffer putKey:@"videocodecid" stringValue:@"avc1"];
    [buffer putStringValue:@"trackinfo"];
    [buffer appendByte:kAMFStrictArray];
    [buffer appendByte32:2];
    
    // Audio stream metadata
    [buffer appendByte:kAMFObject];
    [buffer putKey:@"type" stringValue:@"audio"];
    NSString *desc = [NSString stringWithFormat:@"{AACFrame: codec:AAC, channels:%d, frequency:%f, samplesPerFrame:1024, objectType:LC}",self.config.stereo+1,self.config.audioSampleRate];
    [buffer putKey:@"description" stringValue:desc];
    [buffer putKey:@"timescale" doubleValue:1000.];
    [buffer putStringValue:@"sampledescription"];
    [buffer appendByte:kAMFStrictArray];
    [buffer appendByte32:1];
    [buffer appendByte:kAMFObject];
    [buffer putKey:@"sampletype" stringValue:@"mpeg4-generic"];
    [buffer appendByte:0];
    [buffer appendByte:0];
    [buffer appendByte:kAMFObjectEnd];
    
    [buffer putKey:@"language" stringValue:@"eng"];
    [buffer appendByte:0];
    [buffer appendByte:0];
    [buffer appendByte:kAMFObjectEnd];
    
    // Video stream metadata
    [buffer appendByte:kAMFObject];
    [buffer putKey:@"type" stringValue:@"video"];
    [buffer putKey:@"timescale" doubleValue:1000.];
    [buffer putKey:@"language" stringValue:@"eng"];
    [buffer putStringValue:@"sampledescription"];
    [buffer appendByte:kAMFStrictArray];
    [buffer appendByte32:1];
    [buffer appendByte:kAMFObject];
    [buffer putKey:@"sampletype" stringValue:@"H264"];
    [buffer appendByte:0];
    [buffer appendByte:0];
    [buffer appendByte:kAMFObjectEnd];
    [buffer appendByte:0];
    [buffer appendByte:0];
    [buffer appendByte:kAMFObjectEnd];
    
    [buffer appendByte16:0];
    [buffer appendByte:kAMFObjectEnd];
    [buffer putKey:@"audiodatarate" doubleValue:131152. / 1024.];
    [buffer putKey:@"audiosamplerate" doubleValue:self.config.audioSampleRate];
    [buffer putKey:@"audiosamplesize" doubleValue:16];
    [buffer putKey:@"audiochannels" doubleValue:self.config.stereo + 1];
    [buffer putKey:@"audiocodecid" stringValue:@"mp4a"];
    
    [buffer appendByte:0];
    [buffer appendByte:kAMFObjectEnd];
    
    metadata.msg_type_id = FLV_TAG_TYPE_META;
    metadata.msg_stream_id = SGStreamIDAudio;
    metadata.msg_length.data = (int)buffer.length;
    metadata.timestamp.data = 0;
    
    [self sendPacket:buffer :metadata];
}

//验证过
- (void)sendSetChunkSize:(int32_t)newChunkSize{

    dispatch_sync(_packetQueue, ^{
        int streamId = 0;
        NSMutableData *data = [NSMutableData data];
        [data appendByte:2];
        [data appendByte24:0];
        [data appendByte24:4];
        [data appendByte:SGMSGTypeID_CHUNK_SIZE];
        
        [data appendBytes:(uint8_t*)&streamId length:sizeof(int32_t)];
        [data appendByte32:newChunkSize];
        
        [self writeData:data];
        //这里重新赋值了 16384
        _outChunkSize = newChunkSize;
    });
}

- (void)sendSetBufferTime:(int)milliseconds{
    dispatch_sync(_packetQueue, ^{
        int streamId = 0;
     
        NSMutableData *data = [NSMutableData data];
        [data appendByte:2];
        [data appendByte24:0];
        [data appendByte24:10];
        [data appendByte:SGMSGTypeID_PING];
        [data appendBytes:(uint8_t*)&streamId length:sizeof(int32_t)];
        
        [data appendByte16:3];
        [data appendByte32:_streamID];
        [data appendByte32:milliseconds];
        
        [self writeData:data];
    });
}

- (void)sendPublish{
    RTMPChunk_0 metadata = {0};
    metadata.msg_stream_id = SGStreamIDAudio;
    metadata.msg_type_id = SGMSGTypeID_INVOKE;
    
    NSMutableData *buff = [NSMutableData data];
    [buff appendString:@"publish"];
    [buff appendDouble:++_numOfInvokes];
    self.trackedCommands[@(_numOfInvokes)] = @"publish";
    [buff appendByte:kAMFNull];
    [buff appendString:_url.playPath];
    [buff appendString:@"live"];
    
    metadata.msg_length.data = (int)buff.length;
    [self sendPacket:buff :metadata];
}

- (void)sendCreateStream{
    RTMPChunk_0 metadata = {0};
    metadata.msg_stream_id = SGStreamIDInvoke;
    metadata.msg_type_id = SGMSGTypeID_INVOKE;
    
    NSMutableData *buff = [NSMutableData data];
    [buff appendString:@"createStream"];
    self.trackedCommands[@(++_numOfInvokes)] = @"createStream";
    [buff appendDouble:_numOfInvokes];
    [buff appendByte:kAMFNull];
    
    metadata.msg_length.data = (int)buff.length;
    [self sendPacket:buff :metadata];
}

//未调用
- (void)sendFCUnpublish{
//    RTMPChunk_0 metadata = {0};
//    metadata.msg_stream_id = SGStreamIDInvoke;
//    metadata.msg_type_id = SGMSGTypeID_INVOKE;
//    
//    NSMutableData *buff = [NSMutableData data];
//    [buff appendString:@"FCUnublish"];
//    [buff appendDouble:(++_numOfInvokes)];
//    self.trackedCommands[@(_numOfInvokes)] = @"FCUnublish";
//    [buff appendByte:kAMFNull];
//    [buff appendString:_url.playPath];
//    metadata.msg_length.data = (int)buff.length;
//    
//    [self sendPacket:buff :metadata];
}

- (void)sendFCPublish{
    RTMPChunk_0 metadata = {0};
    metadata.msg_stream_id = SGStreamIDInvoke;
    metadata.msg_type_id = SGMSGTypeID_NOTIFY;
    
    NSMutableData *buff = [NSMutableData data];
    [buff appendString:@"FCPublish"];
    [buff appendDouble:(++_numOfInvokes)];
    self.trackedCommands[@(_numOfInvokes)] = @"FCPublish";
    [buff appendByte:kAMFNull];
    [buff appendString:_url.playPath];
    metadata.msg_length.data = (int)buff.length;
    
    [self sendPacket:buff :metadata];
}

- (void)sendDeleteStream{
    RTMPChunk_0 metadata = {0};
    metadata.msg_stream_id = SGStreamIDInvoke;
    metadata.msg_type_id = SGMSGTypeID_INVOKE;
    
    NSMutableData *buff = [NSMutableData data];
    [buff appendString:@"deleteStream"];
    [buff appendDouble:++_numOfInvokes];
    self.trackedCommands[@(_numOfInvokes)] = @"deleteStream";
    [buff appendByte:kAMFNull];
    [buff appendDouble:_streamID];

    metadata.msg_length.data = (int)buff.length;
    [self sendPacket:buff :metadata];
}

- (void)sendReleaseStream{
    
    RTMPChunk_0 metadata = {0};
    metadata.msg_stream_id = SGStreamIDInvoke;
    metadata.msg_type_id = SGMSGTypeID_NOTIFY;
    
    NSMutableData *buff = [NSMutableData data];
    [buff appendString:@"releaseStream"];
    [buff appendDouble:++_numOfInvokes];
    
    self.trackedCommands[@(_numOfInvokes)] = @"releaseStream";
    [buff appendByte:kAMFNull];
    [buff appendString:_url.playPath];
    
    metadata.msg_length.data = (int)buff.length;
    [self sendPacket:buff :metadata];
}

- (void)writeData:(NSData *)data{
    if (data.length == 0) {
        return;
    }
    
    [self.session writeData:data];

}
@end

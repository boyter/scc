-- lua test

local function f(a)
    for i = 1, a do
        io.write(a .. " x " .. i .. " = " .. (i*a))
        if i ~= a then
            io.write(', ')
        end
    end
    print()
end

for i = 1, 9 do
    f(i)
end

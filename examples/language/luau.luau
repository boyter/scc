-- test for luau

local function f(a: number): string
    local res = ""
    for i = 1, a do
        res = res .. (a .. " x " .. i .. " = " .. (i*a))
        if i ~= a then
            res = res .. (', ')
        end
    end
    return res
end

for i = 1, 9 do
    print(f(i))
end

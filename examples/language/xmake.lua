set_project("test-project")

set_version("0.1.0")

set_xmakever("2.9.9")

set_allowedplats("windows")
set_allowedmodes("debug", "release", "releasedbg")

includes("src", "xmake", "test")

-- fixed config
set_languages("c++20")
add_rules("mode.debug", "mode.release", "mode.releasedbg")

if is_mode("debug") then
    add_defines("LIBRARY_DEBUG")
elseif is_mode("release") then
    set_optimize("smallest")
end

if is_plat("windows") then
    add_defines("UNICODE", "_UNICODE")
    add_cxflags("/permissive-", {tools = "cl"})
end

set_encodings("utf-8")

-- dynamic config
if has_config("dev") then
    set_policy("compatibility.version", "3.0")

    set_warnings("all")

    if is_plat("windows") then
        set_runtimes("MD")
    end
end

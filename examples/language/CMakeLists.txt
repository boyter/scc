cmake_minimum_required(VERSION 3.20)
project(test_project) # project name

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
if (${CMAKE_CXX_COMPILER_ID} MATCHES "Clang" OR ${CMAKE_CXX_COMPILER_ID} MATCHES "GNU" #[[GCC]])
    add_compile_options(-Wall -Wextra)
endif()

#[[
Set special flags when we are using clang toolchain
]]
if(${CMAKE_CXX_COMPILER_ID} STREQUAL Clang)
    if(${CMAKE_CXX_COMPILER_VERSION} VERSION_GREATER "15.0.0" )
        add_compile_definitions(HAS_CLANG_SOURCE_LOCATION)
    endif()
endif()

include_directories("${CMAKE_CURRENT_SOURCE_DIR}/include")
add_executable(test_project main.cpp helper.cpp)

include(CheckIPOSupported)
check_ipo_supported(RESULT ipo_supported OUTPUT output)
if(ipo_supported)
    set_property(TARGET test_project PROPERTY INTERPROCEDURAL_OPTIMIZATION TRUE)
else()
    message(WARNING "IPO is not supported: ${output}")
endif()

/*

 Copyright (C) 2016 Apple Inc. All Rights Reserved.

 See LICENSE.txt for this sample’s licensing information



 Abstract:

 A shader using random noise to create a wood texture for the Metal Shader Showcase. This is an example of a 3D procedural texture based shader. The wood texture is accomplished by making rings of two different colors and using perlin noise to add some variation to the rings.

 */



#include <metal_stdlib>

#include <metal_common>

#include <simd/simd.h>

#include "AAPLSharedTypes.h"



using namespace metal;





struct ColorInOut {

    float4 position [[position]];

    float3 normal_cameraspace;

    float3 eye_direction_cameraspace;

    float3 light_direction_cameraspace;

    float3 position_modelspace;

    float3 position_cameraspace;

};



float rand(int x, int y, int z);

float smoothNoise(float x, float y, float z);

float noise3D(float unscaledX, float unscaledY, float unscaledZ);

float3 woodColor(float3 position);



// Global constants

constant float3 light_position = float3(-1.0, 1.0, -1.0);

constant float4 light_color = float4(1.0, 1.0, 1.0, 1.0);

constant float teapotMin = -0.144000;

constant float teapotMax = 0.164622;

constant float scaleLength = teapotMax - teapotMin;

constant uint NOISE_DIM = 512;

constant float NOISE_SIZE = 64;

constant float3 darkBrown = float3(0.234f, 0.125f, 0.109f);

constant float3 lightBrown = float3(0.168f, 0.133f, 0.043f);

constant float numberOfRings = 84.0;

constant float turbulence = 0.015;

constant float PI = 3.14159;

constant float  materialShine = 50.0;





// Generate a random float in the range [0.0f, 1.0f] using x, y, and z (based on the xor128 algorithm)

float rand(int x, int y, int z)

{

    int seed = x + y * 57 + z * 241;

    seed= (seed<< 13) ^ seed;

    return (( 1.0 - ( (seed * (seed * seed * 15731 + 789221) + 1376312589) & 2147483647) / 1073741824.0f) + 1.0f) / 2.0f;

}



// Return the interpolated noise for the given x, y, and z values. This is done by finding the whole

// number before and after the given position in each dimension. Using these values we can get 6 vertices

// that represent a cube that surrounds the position. We get each of the vertices noise values, and using the

// given position, interpolate between the noise values of the vertices to get the smooth noise.

float smoothNoise(float x, float y, float z)

{

    // Get the truncated x, y, and z values

    int intX = x;

    int intY = y;

    int intZ = z;



    // Get the fractional reaminder of x, y, and z

    float fractX = x - intX;

    float fractY = y - intY;

    float fractZ = z - intZ;



    // Get first whole number before

    int x1 = (intX + NOISE_DIM) % NOISE_DIM;

    int y1 = (intY + NOISE_DIM) % NOISE_DIM;

    int z1 = (intZ + NOISE_DIM) % NOISE_DIM;



    // Get the number after

    int x2 = (x1 + NOISE_DIM - 1) % NOISE_DIM;

    int y2 = (y1 + NOISE_DIM - 1) % NOISE_DIM;

    int z2 = (z1 + NOISE_DIM - 1) % NOISE_DIM;



    // Tri-linearly interpolate the noise

    float sumY1Z1 = mix(rand(x2,y1,z1), rand(x1,y1,z1), fractX);

    float sumY1Z2 = mix(rand(x2,y1,z2), rand(x1,y1,z2), fractX);

    float sumY2Z1 = mix(rand(x2,y2,z1), rand(x1,y2,z1), fractX);

    float sumY2Z2 = mix(rand(x2,y2,z2), rand(x1,y2,z2), fractX);



    float sumZ1 = mix(sumY2Z1, sumY1Z1, fractY);

    float sumZ2 = mix(sumY2Z2, sumY1Z2, fractY);



    float value = mix(sumZ2, sumZ1, fractZ);



    return value;

}



// Generate perlin noise for the given input values. This is done by generating smooth noise at mutiple

// different sizes and adding them together.

float noise3D(float unscaledX, float unscaledY, float unscaledZ)

{

    // Scale the values to force them in the range [0, NOISE_DIM]

    float x = ((unscaledX - teapotMin) / scaleLength) * NOISE_DIM;

    float y = ((unscaledY - teapotMin) / scaleLength) * NOISE_DIM;

    float z = ((unscaledZ - teapotMin) / scaleLength) * NOISE_DIM;



    float value = 0.0f, size = NOISE_SIZE, div = 0.0;



    //Add together smooth noise of increasingly smaller size.

    while(size >= 1.0f)

    {

        value += smoothNoise(x / size, y / size, z / size) * size;

        div += size;

        size /= 2.0f;

    }

    value /= div;



    return value;

}



// Calculate the wood color given the position

float3 woodColor(float3 position)

{

    float x = position.x, y = position.y, z = position.z;



    // Get the distance of the point from the y-axis to identify whether it will be a ring or not.

    // Get the smooth value for that point to add some randomness to the rings and scale the

    // randomness by a factor called turbulence. Use the cosine function to make the rings and

    // interpolate between the two wood ring colors.

    float distanceValue = sqrt(x*x + z*z) + turbulence * noise3D(x, y, z);

    float cosineValue = fabs(cos(2.0f * numberOfRings * distanceValue * PI));



    float3 finalColor = darkBrown + cosineValue * lightBrown;

    return finalColor;

}



// Wood vertex shader function

vertex ColorInOut wood_vertex(device packed_float3* vertices [[ buffer(0) ]],

                              device packed_float3* normals [[ buffer(1) ]],

                              constant AAPL::uniforms_t& uniforms [[ buffer(2) ]],

                              unsigned int vid [[ vertex_id ]])

{

    ColorInOut out;



    float4x4 model_matrix = uniforms.model_matrix;

    float4x4 view_matrix = uniforms.view_matrix;

    float4x4 projection_matrix = uniforms.projection_matrix;

    float4x4 mvp_matrix = projection_matrix * view_matrix * model_matrix;

    float4x4 model_view_matrix = view_matrix * model_matrix;



    // Calculate the position of the object from the perspective of the camera

    float4 vertex_position_modelspace = float4(float3(vertices[vid]), 1.0f);

    out.position = mvp_matrix * vertex_position_modelspace;

    out.position_modelspace = vertices[vid];



    // Calculate the normal from the perspective of the camera

    float3 normal = normals[vid];

    out.normal_cameraspace = (normalize(model_view_matrix * float4(normal, 0.0f))).xyz;



    // Calculate the view vector from the perspective of the camera

    float3 vertex_position_cameraspace = ( view_matrix * model_matrix * vertex_position_modelspace ).xyz;

    out.eye_direction_cameraspace = float3(0.0f,0.0f,0.0f) - vertex_position_cameraspace;



    // Calculate the direction of the light from the position of the camera

    float3 light_position_cameraspace = ( view_matrix * float4(light_position,1.0f)).xyz;

    out.light_direction_cameraspace = light_position_cameraspace + out.eye_direction_cameraspace;



    return out;

}



// Wood fragment shader function

fragment half4 wood_fragment(ColorInOut in [[stage_in]])

{

    half4 color(1.0f);



    // Get the woods base color using the woodColor function

    float3 baseColor = woodColor(in.position_modelspace);



    // Generate material ambient, difuse, and specular colors derived from the base color of the wood

    float3 material_ambient_color = 0.5f * baseColor;

    float3 material_diffuse_color = baseColor;

    float3 material_specular_color = float3(0.4f);



    // Calculate the ambient color

    float3 ambient_component = material_ambient_color;



    // Calculate the diffuse color

    float3 n = normalize(in.normal_cameraspace);

    float3 l = normalize(in.light_direction_cameraspace);

    float n_dot_l = saturate( dot(n, l) );



    float3 diffuse_component = light_color.xyz * n_dot_l * material_diffuse_color;



    // Calculate the specular color

    float3 e = normalize(in.eye_direction_cameraspace);

    float3 r = -l + 2.0f * n_dot_l * n;

    float e_dot_r =  saturate( dot(e, r) );

    float3 specular_component = material_specular_color * light_color.xyz * pow(e_dot_r, materialShine);



    // Combine the ambient, specular and diffuse colors to get the final color

    color.rgb = half3(ambient_component + diffuse_component + specular_component);



    return color;

};
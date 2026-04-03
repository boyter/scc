#ifndef VECTOR_OPS_CUH
#define VECTOR_OPS_CUH

// An example of CUDA Header files

#include <cuda_runtime.h>

#define CHECK_CUDA(call) \
    if ((call) != cudaSuccess) { \
        printf("CUDA Error at %s:%d\n", __FILE__, __LINE__); \
    }

// Kernel definition
__global__ void vectorAddKernel(const float* A, const float* B, float* C, int n);

void launchVectorAdd(const float* h_A, const float* h_B, float* h_C, int n);

#endif

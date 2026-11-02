// Define the structure for vertex shader outputs / fragment shader inputs
struct VertexOutput {
    @builtin(position) position: vec4f,
    @location(0) color: vec4f,
}

@vertex
fn vs_main(@builtin(vertex_index) idx: u32) -> VertexOutput {
    let vertices = array<vec2f, 3>(
        vec2f(-0.5, -0.5),
        vec2f( 0.5, -0.5),
        vec2f( 0.0,  0.5)
    );

    /*
      different colors
      /* a nested comment */
    */
    let colors = array<vec4f, 3>(
        vec4f(1.0, 0.0, 0.0, 1.0), // bottom left: red
        vec4f(0.0, 1.0, 0.0, 1.0), // bottom right: green
        vec4f(0.0, 0.0, 1.0, 1.0)  // top: blue
    );

    var output: VertexOutput;
    output.position = vec4f(vertices[idx], 0.0, 1.0);
    output.color = colors[idx];
    return output;
}

@fragment
fn main(in: VertexOutput) -> @location(0) vec4f {
    return in.color;
}

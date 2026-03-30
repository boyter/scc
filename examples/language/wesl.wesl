import package::{color::palette, shape::star};
import lygia::{draw::fill::fill, generative::snoise::snoise2};
import env::u;

// this is a comment
@fragment
fn main(@builtin(position) pos: vec4f) -> @location(0) vec4f {
  let uv = pos.xy / u.resolution;
  var sdf = star(pos.xy, u.resolution);

  /*
  this is also a comment
  */
  @if(noise)  // conditions at runtime or build time
  sdf += snoise2(uv * 8.0 + u.time) * 0.15;

  let shape = fill(sdf, 0.6);
  return vec4f(palette(uv, u.time) * shape, shape);
}
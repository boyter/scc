// Copyright 2018 Schuyler Eldridge
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

`timescale 1ns / 1ps
module button_debounce
  #(
    parameter
    CLK_FREQUENCY  = 10_000_000,
    DEBOUNCE_HZ    = 2
    // These parameters are specified such that you can choose any power
    // of 2 frequency for a debouncer between 1 Hz and
    // CLK_FREQUENCY. Note, that this will throw errors if you choose a
    // non power of 2 frequency (i.e. count_value evaluates to some
    // number / 3 which isn't interpreted as a logical right shift). I'm
    // assuming this will not work for DEBOUNCE_HZ values less than 1,
    // however, I'm uncertain of the value of a debouncer for fractional
    // hertz button presses.
    )
  (
   input      clk,     // clock
   input      reset_n, // asynchronous reset
   input      button,  // bouncy button
   output reg debounce // debounced 1-cycle signal
   );

  localparam
    COUNT_VALUE  = CLK_FREQUENCY / DEBOUNCE_HZ,
    WAIT         = 0,
    FIRE         = 1,
    COUNT        = 2;

  reg [1:0]   state, next_state;
  reg [25:0]  count;

  always @ (posedge clk or negedge reset_n)
    state <= (!reset_n) ? WAIT : next_state;

  always @ (posedge clk or negedge reset_n) begin
    if (!reset_n) begin
      debounce <= 0;
      count    <= 0;
    end
    else begin
      debounce <= 0;
      count    <= 0;
      case (state)
        WAIT: begin
        end
        FIRE: begin
          debounce <= 1;
        end
        COUNT: begin
          count <= count + 1;
        end
      endcase
    end
  end

  always @ * begin
    case (state)
      WAIT:    next_state = (button)                  ? FIRE : state;
      FIRE:    next_state = COUNT;
      COUNT:   next_state = (count > COUNT_VALUE - 1) ? WAIT : state;
      default: next_state = WAIT;
    endcase
  end

endmodule

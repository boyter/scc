#!/usr/bin/env python3
"""Convert hyperfine JSON output from benchmark.sh into a Google Charts HTML page."""

import json
import re
import sys


def extract_version(command):
    """Extract version from a command like 'scc3.4.0 linux'."""
    m = re.match(r'scc(\d+\.\d+\.\d+)', command)
    return m.group(1) if m else command


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} benchmark_regression.json [title]")
        sys.exit(1)

    json_file = sys.argv[1]
    title = sys.argv[2] if len(sys.argv) > 2 else "scc performance linux kernel"

    with open(json_file) as f:
        data = json.load(f)

    rows = []
    for result in data["results"]:
        version = extract_version(result["command"])
        mean = result["mean"]
        rows.append(f"          ['{version}', {mean:.3f}]")

    chart_data = ",\n".join(rows)

    html = f"""<!DOCTYPE html>
<html>
<head>
  <script type="text/javascript" src="https://www.gstatic.com/charts/loader.js"></script>
  <script type="text/javascript">
    google.charts.load('current', {{'packages':['corechart']}});
    google.charts.setOnLoadCallback(drawChart);

    function drawChart() {{
      var data = google.visualization.arrayToDataTable([
        ['Version', 'Runtime (seconds)'],
{chart_data}
      ]);

      var options = {{
        title: '{title}',
        curveType: 'function',
        legend: {{ position: 'bottom' }}
      }};

      var chart = new google.visualization.LineChart(document.getElementById('curve_chart'));
      chart.draw(data, options);
    }}
  </script>
</head>
<body>
  <div id="curve_chart" style="width: 900px; height: 500px"></div>
</body>
</html>
"""

    print(html)


if __name__ == "__main__":
    main()

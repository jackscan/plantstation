window.onload = function () {
    var horizonalLinePlugin = {
        beforeDraw: function (chartInstance) {
            var yValue;
            var yScale = chartInstance.scales["weight-y-axis"];
            var canvas = chartInstance.chart;
            var ctx = canvas.ctx;
            var index;
            var line;
            var style;

            ctx.save();
            ctx.lineWidth = 1;
            ctx.setLineDash([5, 3]);

            if (chartInstance.options.horizontalLine) {
                for (index = 0; index < chartInstance.options.horizontalLine.length; index++) {
                    line = chartInstance.options.horizontalLine[index];

                    if (!line.style || !line.y)
                        continue;

                    style = line.style;
                    yValue = yScale.getPixelForValue(line.y);


                    if (yValue) {
                        ctx.beginPath();
                        ctx.moveTo(chartInstance.chartArea.left, yValue);
                        ctx.lineTo(chartInstance.chartArea.right, yValue);
                        ctx.strokeStyle = style;
                        ctx.stroke();
                    }

                    if (line.text) {
                        ctx.fillStyle = style;
                        ctx.fillText(line.text, 0, yValue + ctx.lineWidth);
                    }
                }
            }
            ctx.restore();
        }
    };
    Chart.pluginService.register(horizonalLinePlugin);

    var chart = new Chart(document.getElementById("wchart"), {
        type: 'bar',
        data: {
            labels: [],
            datasets: [
                {
                    type: 'line',
                    data: [],
                    label: "Plant Weight 1",
                    yAxisID: 'weight-y-axis',
                    borderColor: "#205020",
                    backgroundColor: "#408040",
                    fill: false
                },
                {
                    type: 'line',
                    data: [],
                    label: "Plant Weight 2",
                    yAxisID: 'weight-y-axis',
                    borderColor: "#306030",
                    backgroundColor: "#509050",
                    fill: false
                },
                {
                    type: 'line',
                    data: [],
                    label: "Temperature",
                    yAxisID: 'temp-y-axis',
                    borderColor: "#ff8000",
                    backgroundColor: "#ffa000",
                    fill: false
                },
                {
                    type: 'line',
                    data: [],
                    label: "Humidity",
                    yAxisID: 'hum-y-axis',
                    borderColor: "#2080ff",
                    backgroundColor: "#3090ff",
                    fill: false
                },
                {
                    type: 'line',
                    data: [],
                    label: "Average Weight 1",
                    yAxisID: 'weight-y-axis',
                    borderColor: "#ffa000",
                    backgroundColor: "#ffc040",
                    borderWidth: 1,
                    pointRadius: 0,
                    fill: false
                },
                {
                    type: 'line',
                    data: [],
                    label: "Average Weight 2",
                    yAxisID: 'weight-y-axis',
                    borderColor: "#ffb010",
                    backgroundColor: "#ffd050",
                    borderWidth: 1,
                    pointRadius: 0,
                    fill: false
                },
                {
                    type: 'line',
                    data: [],
                    label: "Water Level",
                    yAxisID: 'level-y-axis',
                    borderColor: "#001080",
                    backgroundColor: "#0020ff",
                    fill: false
                },
                {
                    type: 'bar',
                    data: [],
                    yAxisID: 'water-y-axis',
                    label: "Watering 1",
                    borderColor: "#0030a0",
                    backgroundColor: "#1060c0",
                    fill: false
                },
                {
                    type: 'bar',
                    data: [],
                    yAxisID: 'water-y-axis',
                    label: "Watering 2",
                    borderColor: "#0030a0",
                    backgroundColor: "#1060c0",
                    fill: false
                }
            ]
        },
        options: {
            horizontalLine: [],
            elements: {
                line: {
                    cubicInterpolationMode: 'monotone'
                }
            },
            scales: {
                xAxes: [{
                    id: 'hour-x-axis',
                    offset: false,
                    ticks: {
                        maxTicksLimit: 48,
                        maxRotation: 0
                    },
                    gridLines: {
                        offsetGridLines: false
                    }
                }],
                yAxes: [{
                    id: 'water-y-axis',
                    type: 'linear',
                    position: 'left'
                }, {
                    id: 'level-y-axis',
                    type: 'linear',
                    position: 'left'
                }, {
                    id: 'weight-y-axis',
                    type: 'linear',
                    position: 'right'
                }, {
                    id: 'temp-y-axis',
                    type: 'linear',
                    position: 'right',
                    ticks: {
                        suggestedMin: 10,
                        suggestedMax: 30
                    }
                }, {
                    id: 'hum-y-axis',
                    type: 'linear',
                    position: 'right',
                    ticks: {
                        suggestedMin: 0,
                        suggestedMax: 100
                    }
                }]
            }
        }
    });

    var minchart = new Chart(document.getElementById("minchart"), {
        type: 'bar',
        data: {
            labels: [],
            datasets: [
                {
                    type: 'line',
                    data: [],
                    label: "Plant Weight 1",
                    yAxisID: 'weight-y-axis',
                    borderColor: "#205020",
                    backgroundColor: "#408040",
                    fill: false
                },
                {
                    type: 'line',
                    data: [],
                    label: "Plant Weight 2",
                    yAxisID: 'weight-y-axis',
                    borderColor: "#306030",
                    backgroundColor: "#509050",
                    fill: false
                },
                {
                    type: 'line',
                    data: [],
                    label: "Temperature",
                    yAxisID: 'temp-y-axis',
                    borderColor: "#ff8000",
                    backgroundColor: "#ffa000",
                    fill: false
                },
                {
                    type: 'line',
                    data: [],
                    label: "Humidity",
                    yAxisID: 'hum-y-axis',
                    borderColor: "#2080ff",
                    backgroundColor: "#3090ff",
                    fill: false
                },
                {
                    type: 'line',
                    data: [],
                    label: "Water Level",
                    yAxisID: 'level-y-axis',
                    borderColor: "#001080",
                    backgroundColor: "#0020ff",
                    fill: false
                }
            ]
        },
        options: {
            elements: {
                line: {
                    cubicInterpolationMode: 'monotone'
                }
            },
            scales: {
                xAxes: [{
                    id: 'min-x-axis',
                    offset: false,
                    ticks: {
                        maxTicksLimit: 48,
                        maxRotation: 0
                    },
                    gridLines: {
                        offsetGridLines: false
                    }
                }],
                yAxes: [{
                    id: 'level-y-axis',
                    type: 'linear',
                    position: 'left',
                    ticks: {
                        suggestedMin: 500,
                        suggestedMax: 1000
                    }
                }, {
                    id: 'weight-y-axis',
                    type: 'linear',
                    position: 'right'
                }, {
                    id: 'temp-y-axis',
                    type: 'linear',
                    position: 'right',
                    ticks: {
                        suggestedMin: 10,
                        suggestedMax: 30
                    }
                }, {
                    id: 'hum-y-axis',
                    type: 'linear',
                    position: 'right',
                    ticks: {
                        suggestedMin: 0,
                        suggestedMax: 100
                    }
                }]
            }
        }
    });

    function getData() {
        var xhttp = new XMLHttpRequest();
        xhttp.onreadystatechange = function () {
            if (this.readyState == 4 && this.status == 200) {
                var resp = JSON.parse(xhttp.responseText);
                var data = resp.data;
                var len = Math.max(data.weight[0].length, data.weight[1].length);
                var start = (data.time + 1 - (len % 24) + 24) % 24;
                var iw = 0;
                var h;
                var avg1 = 0;
                var avg2 = 0;
                var count1 = 0;
                var count2 = 0;
                var i, j, w1, w2;
                for (i = 0; i < len; ++i) {
                    w1 = data.water[0][i];
                    w2 = data.water[1][i];
                    h = (start + i) % 24;
                    chart.data.labels.push(h);
                    // chart.data.datasets[0].data.push(data.moisture[i]);
                    // 4052 is weight value with no load
                    chart.data.datasets[0].data.push(data.weight[0][i]);
                    chart.data.datasets[1].data.push(data.weight[1][i]);
                    chart.data.datasets[2].data.push(data.temperature[i] / 100);
                    chart.data.datasets[3].data.push(data.humidity[i] / 100);
                    chart.data.datasets[7].data.push(w1 / 1000);
                    chart.data.datasets[8].data.push(w2 / 1000);
                    avg1 += data.weight[0][i];
                    avg2 += data.weight[1][i];
                    ++count1;
                    ++count2;
                    if (w1 > 0) {
                        // fill average data
                        avg1 /= count1;
                        for (j = 0; j < count1; ++j)
                            chart.data.datasets[4].data.push(avg1);
                        avg1 = 0;
                        count1 = 0;
                    }
                    if (w2 > 0) {
                        // fill average data
                        avg2 /= count2;
                        for (j = 0; j < count2; ++j)
                            chart.data.datasets[5].data.push(avg2);
                        avg2 = 0;
                        count2 = 0;
                    }
                }

                if (count1 > 0) {
                    avg1 /= count1;
                    for (j = 0; j < count1; ++j)
                        chart.data.datasets[4].data.push(avg1);
                }

                if (count2 > 0) {
                    avg2 /= count2;
                    for (j = 0; j < count2; ++j)
                        chart.data.datasets[5].data.push(avg2);
                }

                var col = [
                    { range: '#c0c0c0', low: '#ff0000', dst: '#40b000' },
                    { range: '#d0d0d0', low: '#ff1010', dst: '#50c010' },
                ];

                resp.config.forEach(function(config, i) {
                    chart.options.scales.yAxes[0].ticks.min = 0;
                    chart.options.scales.yAxes[0].ticks.max = Math.ceil(config.max / 1000);
                    chart.options.scales.yAxes[2].ticks.suggestedMin = Math.floor((config.low - config.range * 2) / 10) * 10;
                    chart.options.scales.yAxes[2].ticks.suggestedMax = Math.ceil((config.dst + config.range * 2) / 10) * 10;

                    chart.options.horizontalLine.push({y: config.dst-config.range, style: col[i].range});
                    chart.options.horizontalLine.push({y: config.dst+config.range, style: col[i].range});
                    chart.options.horizontalLine.push({y: config.low, style: col[i].low});
                    chart.options.horizontalLine.push({y: config.dst, style: col[i].dst});
                });

                chart.update();

                var mindata = resp.mindata;
                var mlen = Math.max(mindata.weight[0].length, mindata.weight[1].length);
                var minstart = (mindata.time + 1 - (mlen % 60) + 60) % 60;
                var min;
                for (i = 0; i < mlen; ++i) {
                    min = (minstart + i) % 60;
                    minchart.data.labels.push(min);
                    // minchart.data.datasets[0].data.push(mindata.moisture[i]);
                    // 4052 is weight value with no load
                    minchart.data.datasets[0].data.push(mindata.weight[0][i]);
                    minchart.data.datasets[1].data.push(mindata.weight[1][i]);
                    minchart.data.datasets[2].data.push(mindata.temperature[i] / 100);
                    minchart.data.datasets[3].data.push(mindata.humidity[i] / 100);
                }

                minchart.update();
            }
        };
        xhttp.open("GET", "/data", true);
        xhttp.send();
    }
    getData();
};

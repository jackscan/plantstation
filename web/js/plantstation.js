window.onload = function () {
    var horizonalLinePlugin = {
        beforeDraw: function (chartInstance) {
            var yValue;
            var yScale = chartInstance.scales["moist-y-axis"];
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
                    yAxisID: 'moist-y-axis',
                    label: "Moisture",
                    borderColor: "#30a000",
                    backgroundColor: "#60c010",
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
                    label: "Average Moisture",
                    yAxisID: 'moist-y-axis',
                    borderColor: "#ffa000",
                    backgroundColor: "#ffc040",
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
                    type: 'line',
                    data: [],
                    label: "Plant Weight",
                    yAxisID: 'weight-y-axis',
                    borderColor: "#205020",
                    backgroundColor: "#408040",
                    fill: false
                },
                {
                    type: 'bar',
                    data: [],
                    yAxisID: 'water-y-axis',
                    label: "Watering",
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
                    id: 'moist-x-axis',
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
                    id: 'moist-y-axis',
                    type: 'linear',
                    position: 'left'
                }, {
                    id: 'water-y-axis',
                    type: 'linear',
                    position: 'right'
                }, {
                    id: 'level-y-axis',
                    type: 'linear',
                    position: 'right'
                }, {
                    id: 'weight-y-axis',
                    type: 'linear',
                    position: 'left'
                }, {
                    id: 'temp-y-axis',
                    type: 'linear',
                    position: 'left',
                    ticks: {
                        suggestedMin: 10,
                        suggestedMax: 30
                    }
                }, {
                    id: 'hum-y-axis',
                    type: 'linear',
                    position: 'left',
                    ticks: {
                        suggestedMin: 0,
                        suggestedMax: 100
                    }
                }]
            }
        }
    });

    // function getConfig() {
    //     var xhttp = new XMLHttpRequest();
    //     xhttp.onreadystatechange = function () {
    //         if (this.readyState == 4 && this.status == 200) {
    //             var resp = JSON.parse(xhttp.responseText);
    //             var hour = resp[0];
    //             var minw = resp[1]/1000;
    //             var maxw = resp[2]/1000;
    //             var minm = resp[3];
    //             var dstm = resp[4];
    //             var dm = dstm - minm;
    //             chart.options.scales.yAxes[1].ticks.min = Math.floor(minw);
    //             chart.options.scales.yAxes[1].ticks.max = Math.ceil(maxw);
    //             chart.options.scales.yAxes[0].ticks.suggestedMin = Math.floor((minm - dm) / 10) * 10;
    //             chart.options.scales.yAxes[0].ticks.suggestedMax = Math.ceil((dstm + dm * 2) / 10) * 10;

    //             chart.options.horizontalLine.push({y: minm, style: '#ff0000'});
    //             chart.options.horizontalLine.push({y: dstm, style: '#a0a0a0'});

    //             chart.update();
    //         }
    //     };
    //     xhttp.open("GET", "/config", true);
    //     xhttp.send();
    // }

    function getData() {
        var xhttp = new XMLHttpRequest();
        xhttp.onreadystatechange = function () {
            if (this.readyState == 4 && this.status == 200) {
                var resp = JSON.parse(xhttp.responseText);
                var data = resp.data
                var mlen = data.moisture.length;
                var wlen = data.water.length;
                var start = (data.time + 1 - (mlen % 24) + 24) % 24;
                var iw = 0;
                var h;
                var avg = 0;
                var count = 0;
                var i, j, w;
                for (i = 0; i < mlen; ++i) {
                    w = data.water[i];
                    h = (start + i) % 24;
                    chart.data.labels.push(h);
                    chart.data.datasets[0].data.push(data.moisture[i]);
                    chart.data.datasets[1].data.push(data.temperature[i] / 100);
                    chart.data.datasets[2].data.push(data.humidity[i] / 100);
                    chart.data.datasets[4].data.push(data.level[i]);
                    // 4052 is weight value with no load
                    chart.data.datasets[5].data.push(data.weight[i]);
                    chart.data.datasets[6].data.push(w / 1000);
                    avg += data.moisture[i];
                    ++count;
                    if (w > 0) {
                        // fill average data
                        avg /= count;
                        for (j = 0; j < count; ++j)
                            chart.data.datasets[3].data.push(avg);
                        avg = 0;
                        count = 0;
                    }
                }

                if (count > 0) {
                    avg /= count;
                    for (j = 0; j < count; ++j)
                        chart.data.datasets[3].data.push(avg);
                }

                // console.log(chart.data.datasets[0].data.length + ", " + chart.data.datasets[1].data.length);
                // console.log(chart.data.datasets[1].data);

                // read config
                // var hour = resp[0];
                // var minw = resp[1]/1000;
                // var maxw = resp[2]/1000;
                // var minm = resp[3];
                // var dstm = resp[4];
                var config = resp.config;
                chart.options.scales.yAxes[1].ticks.min = 0;
                chart.options.scales.yAxes[1].ticks.max = Math.ceil(config.max / 1000);
                chart.options.scales.yAxes[0].ticks.suggestedMin = Math.floor((config.low - config.range * 2) / 10) * 10;
                chart.options.scales.yAxes[0].ticks.suggestedMax = Math.ceil((config.dst + config.range * 2) / 10) * 10;

                chart.options.horizontalLine.push({y: config.dst-config.range, style: '#d0d0d0'});
                chart.options.horizontalLine.push({y: config.dst+config.range, style: '#d0d0d0'});
                chart.options.horizontalLine.push({y: config.low, style: '#ff0000'});
                chart.options.horizontalLine.push({y: config.dst, style: '#40b000'});

                chart.update();
            }
        };
        xhttp.open("GET", "/data", true);
        xhttp.send();
    }
    getData();
};

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

func main() {
	http.Handle("/", http.HandlerFunc(indexHandler))
	http.Handle("/ws", websocket.Handler(wsHandler))
	http.Handle("/cpuusage", http.HandlerFunc(cpuUsageHandler))
	fmt.Println("Server started. Listening on :8080")
	http.ListenAndServe(":8080", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>	
  <title>WebSocket Demo</title>
  <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
  <script>
    window.onload = function() {
      var socket = new WebSocket("ws://localhost:8080/ws");
      var chartData = {
        labels: Array.from({ length: 10 }, (_, i) => i),
        datasets: [{
          label: 'CPU Usage',
          data: [],
          backgroundColor: 'rgba(255, 99, 132, 0.2)',
          borderColor: 'rgba(255, 99, 132, 1)',
          borderWidth: 1
        }]
      };

      var ctx = document.getElementById('myChart').getContext('2d');
	  var ctx2 = document.getElementById('myChart2').getContext('2d');
      var myChart = new Chart(ctx, {
	   responsive: true,
  maintainAspectRatio: false,
        type: 'line',
        data: chartData,
        options: {
          scales: {
            y: {
			  beginAtZero: true,
			  suggestedMax: 100
            },
            x: {
              ticks: {
                autoSkip: true,
                maxTicksLimit: 10,
              }
            }
          }
        }
      });
	   var myChart2 = new Chart(ctx2, {
	   responsive: true,
  		maintainAspectRatio: false,
        type: 'line',
        data: chartData,
        options: {
          scales: {
            y: {
            },
            x: {
              ticks: {
                autoSkip: true,
                maxTicksLimit: 10,
              }
            }
          }
        }
      });

      socket.onopen = function() {
        console.log("WebSocket connection opened");
        updateGraph();
      };

      	socket.onmessage = function(event) {
        	var data = JSON.parse(event.data);
        	chartData.datasets[0].data.push(data.utilization);
			if (chartData.datasets[0].data.length > 100) {
				chartData.datasets[0].data.shift();
				chartData.labels.shift();
			}
			chartData.labels.push(chartData.labels[chartData.labels.length - 1] + 1);

			updateCpuInfo(data);
			myChart.update();
			myChart2.update();
      };

      socket.onclose = function() {
        console.log("WebSocket connection closed");
      };

      function updateGraph() {
        setTimeout(updateGraph, 1000);
      }

      function updateCpuInfo(data) {
        document.getElementById('cpuUsage').innerText = data.utilization;
        document.getElementById('user').innerText = data.user;
        document.getElementById('nice').innerText = data.nice;
        document.getElementById('system').innerText = data.system;
        document.getElementById('idle').innerText = data.idle;
        document.getElementById('iowait').innerText = data.iowait;
        document.getElementById('irq').innerText = data.irq;
        document.getElementById('softirq').innerText = data.softirq;
        document.getElementById('utilization').innerText = data.utilization;
        document.getElementById('coreCount').innerText = data.coreCount;
      }
    }
  </script>
</head>
<body>
  <h1>CPU Usage Monitor</h1>
 
  <div>
    <p> CPU Usage: <span id="cpuUsage"></span> </p>
    <p> user: <span id="user"></span> (ユーザーモードで消費したCPU時間)</p>
    <p> nice: <span id="nice"></span> (nice 値が正の プロセスが消費したCPU時間)</p>
    <p> system: <span id="system"></span> (カーネルモードで消費したCPU時間)</p>
    <p> idle: <span id="idle"></span> (CPU がアイドル状態だった時間)</p>
    <p> iowait: <span id="iowait"></span> (I/O 待機状態だった時間)</p>
    <p> irq: <span id="irq"></span> (ハードウェア割り込みが処理された時間)</p>
    <p> softirq: <span id="softirq"></span> (ソフトウェア割り込みが処理された時間)</p>
    <p> CPU 使用率: <span id="utilization"></span></p>
    <p> CPU コア数: <span id="coreCount"></span></p>
  </div>
  <div style="width: 40%">
   <canvas id="myChart" width="100" height="50"></canvas>
   <canvas id="myChart2" width="100" height="50"></canvas>

  </div>
</body>
</html>`)
}

func wsHandler(ws *websocket.Conn) {
	defer ws.Close()

	// 5秒ごとにCPU使用率を送信する
	for {
		time.Sleep(500 * time.Millisecond)
		cpuUsage := getCPUUsage()
		if err := websocket.Message.Send(ws, cpuUsage); err != nil {
			fmt.Println("Error sending data:", err)
			return
		}
	}
}

func getCPUUsage() string {
	// /proc/stat ファイルを開く
	file, err := os.Open("/proc/stat")
	if err != nil {
		fmt.Println("Error:", err)
		return "error"
	}
	defer file.Close()

	// ファイルを読み取る
	scanner := bufio.NewScanner(file)

	var totalUser, totalNice, totalSystem, totalIdle, totalIoWait, totalIrq, totalSoftIrq float64
	var cpuCount int

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// "cpu"で始まる行を処理する
		if len(fields) > 0 && strings.HasPrefix(fields[0], "cpu") {
			if fields[0] == "cpu" {
				// CPU 全体の統計情報
				totalUser, _ = strconv.ParseFloat(fields[1], 64)
				totalNice, _ = strconv.ParseFloat(fields[2], 64)
				totalSystem, _ = strconv.ParseFloat(fields[3], 64)
				totalIdle, _ = strconv.ParseFloat(fields[4], 64)
				totalIoWait, _ = strconv.ParseFloat(fields[5], 64)
				totalIrq, _ = strconv.ParseFloat(fields[6], 64)
				totalSoftIrq, _ = strconv.ParseFloat(fields[7], 64)
			} else {
				// 個々のCPUコアの統計情報
				cpuCount++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error:", err)
		return "error"
	}

	// CPU 使用率の計算
	total := totalUser + totalNice + totalSystem + totalIdle + totalIoWait + totalIrq + totalSoftIrq
	cpuUtilization := (total - totalIdle - totalIoWait) / total * 100
	data := map[string]interface{}{
		"user":        totalUser,
		"nice":        totalNice,
		"system":      totalSystem,
		"idle":        totalIdle,
		"iowait":      totalIoWait,
		"irq":         totalIrq,
		"softirq":     totalSoftIrq,
		"utilization": cpuUtilization,
		"coreCount":   cpuCount,
	}

	output, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error:", err)
		return "error"
	}
	return string(output)
}
func cpuUsageHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, getCPUUsage())
}

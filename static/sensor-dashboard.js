// sensor-dashboard.js
// Function to initialize the dashboard
function initDashboard() {
  // Connect to SSE endpoint
  const eventSource = new EventSource("/api/sensor-stream");

  // Keep track of the latest readings for each location
  const latestReadings = {
    "AM300/OUTSIDE": {
      temperature: null,
      humidity: null,
      co2: null,
      timestamp: null,
    },
    "AM300/INSIDE": {
      temperature: null,
      humidity: null,
      co2: null,
      timestamp: null,
    },
  };

  // Display elements
  const outsideTemp = document.getElementById("outside-temp");
  const outsideHumidity = document.getElementById("outside-humidity");
  const outsideCO2 = document.getElementById("outside-co2");
  const outsideTimestamp = document.getElementById("outside-timestamp");

  const insideTemp = document.getElementById("inside-temp");
  const insideHumidity = document.getElementById("inside-humidity");
  const insideCO2 = document.getElementById("inside-co2");
  const insideTimestamp = document.getElementById("inside-timestamp");

  // Chart data arrays
  let tempChartData = [];
  let humidityChartData = [];
  let co2ChartData = [];

  // Initialize charts
  let tempChart, humidityChart, co2Chart;

  // Check if Chart.js is loaded properly before creating charts
  if (typeof Chart !== "undefined") {
    try {
      tempChart = createChart("temp-chart", "Temperature (°C)");
      humidityChart = createChart("humidity-chart", "Humidity (%)");
      co2Chart = createChart("co2-chart", "CO2 (ppm)");
      console.log("Charts initialized successfully");
    } catch (e) {
      console.error("Error initializing charts:", e);
      document.getElementById("temp-chart-error").classList.remove("d-none");
      document
        .getElementById("humidity-chart-error")
        .classList.remove("d-none");
      document.getElementById("co2-chart-error").classList.remove("d-none");
    }
  } else {
    console.error("Chart.js library not loaded. Charts will not be displayed.");
    document.getElementById("temp-chart-error").classList.remove("d-none");
    document.getElementById("humidity-chart-error").classList.remove("d-none");
    document.getElementById("co2-chart-error").classList.remove("d-none");
  }

  // Event listeners
  eventSource.addEventListener("keepalive", function (e) {
    console.log("Connection established with server");
  });

  eventSource.addEventListener("sensor-update", function (e) {
    const data = JSON.parse(e.data);

    // Update latest readings
    if (latestReadings[data.topic]) {
      latestReadings[data.topic] = {
        temperature: data.temperature,
        humidity: data.humidity,
        co2: data.co2,
        timestamp: new Date(data.timestamp),
      };

      // Update UI for the specific location
      updateUI();

      // Update charts
      updateCharts(data);
    }
  });

  eventSource.addEventListener("error", function (e) {
    console.error("SSE connection error:", e);
    // Try to reconnect after 5 seconds
    setTimeout(() => {
      console.log("Attempting to reconnect...");
      eventSource.close();
      const newEventSource = new EventSource("/api/sensor-stream");
      eventSource = newEventSource;
    }, 5000);
  });

  // Load historical data on page load
  loadHistoricalData();

  // Functions
  function updateUI() {
    // Update outside readings
    const outside = latestReadings["AM300/OUTSIDE"];
    if (outside.timestamp) {
      outsideTemp.textContent = outside.temperature.toFixed(1) + "°C";
      outsideHumidity.textContent = outside.humidity.toFixed(1) + "%";
      outsideCO2.textContent = outside.co2.toFixed(0) + " ppm";
      outsideTimestamp.textContent = formatTimestamp(outside.timestamp);
    }

    // Update inside readings
    const inside = latestReadings["AM300/INSIDE"];
    if (inside.timestamp) {
      insideTemp.textContent = inside.temperature.toFixed(1) + "°C";
      insideHumidity.textContent = inside.humidity.toFixed(1) + "%";
      insideCO2.textContent = inside.co2.toFixed(0) + " ppm";
      insideTimestamp.textContent = formatTimestamp(inside.timestamp);
    }
  }

  function updateCharts(data) {
    // Skip chart updates if Chart.js didn't load properly
    if (
      typeof Chart === "undefined" ||
      !tempChart ||
      !humidityChart ||
      !co2Chart
    ) {
      console.warn("Charts not available, skipping update");
      return;
    }

    const time = new Date(data.timestamp);

    // Keep only the last 50 readings for each chart
    if (tempChartData.length > 50) {
      tempChartData.shift();
      humidityChartData.shift();
      co2ChartData.shift();
    }

    // Add new data points (with null checks)
    const insideTemp = latestReadings["AM300/INSIDE"]?.temperature || null;
    const outsideTemp = latestReadings["AM300/OUTSIDE"]?.temperature || null;
    const insideHum = latestReadings["AM300/INSIDE"]?.humidity || null;
    const outsideHum = latestReadings["AM300/OUTSIDE"]?.humidity || null;
    const insideCO2 = latestReadings["AM300/INSIDE"]?.co2 || null;
    const outsideCO2 = latestReadings["AM300/OUTSIDE"]?.co2 || null;

    tempChartData.push({
      time: time,
      inside: insideTemp,
      outside: outsideTemp,
    });

    humidityChartData.push({
      time: time,
      inside: insideHum,
      outside: outsideHum,
    });

    co2ChartData.push({
      time: time,
      inside: insideCO2,
      outside: outsideCO2,
    });

    try {
      // Update the charts
      updateChart(tempChart, tempChartData, "Temperature (°C)");
      updateChart(humidityChart, humidityChartData, "Humidity (%)");
      updateChart(co2Chart, co2ChartData, "CO2 (ppm)");
    } catch (e) {
      console.error("Error updating charts:", e);
    }
  }

  function createChart(canvasId, label) {
    try {
      const canvas = document.getElementById(canvasId);
      if (!canvas) {
        console.error(`Canvas element with ID ${canvasId} not found`);
        return null;
      }

      const ctx = canvas.getContext("2d");
      if (!ctx) {
        console.error(`Could not get 2D context for canvas ${canvasId}`);
        return null;
      }

      return new Chart(ctx, {
        type: "line",
        data: {
          datasets: [
            {
              label: "Inside",
              borderColor: "rgba(75, 192, 192, 1)",
              backgroundColor: "rgba(75, 192, 192, 0.2)",
              tension: 0.1,
              data: [],
            },
            {
              label: "Outside",
              borderColor: "rgba(255, 99, 132, 1)",
              backgroundColor: "rgba(255, 99, 132, 0.2)",
              tension: 0.1,
              data: [],
            },
          ],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          scales: {
            x: {
              type: "time",
              time: {
                unit: "minute",
              },
              title: {
                display: true,
                text: "Time",
              },
            },
            y: {
              title: {
                display: true,
                text: label,
              },
            },
          },
        },
      });
    } catch (e) {
      console.error(`Error creating chart ${canvasId}:`, e);
      return null;
    }
  }

  function updateChart(chart, data, label) {
    // Skip update if chart is not properly initialized
    if (!chart) {
      console.warn(
        `Cannot update chart for ${label}, chart object is not available`,
      );
      return;
    }

    try {
      // Format data for Chart.js
      const formattedData = data.map((item) => ({
        x: item.time,
        y: item.inside,
      }));

      const formattedData2 = data.map((item) => ({
        x: item.time,
        y: item.outside,
      }));

      // Update datasets directly
      chart.data.datasets[0].data = formattedData;
      chart.data.datasets[1].data = formattedData2;

      // Update label
      if (chart.options.scales && chart.options.scales.y) {
        chart.options.scales.y.title.text = label;
      }

      // Update the chart
      chart.update();
    } catch (e) {
      console.error(`Error updating ${label} chart:`, e);
    }
  }

  function loadHistoricalData() {
    fetch("/api/readings")
      .then((response) => response.json())
      .then((data) => {
        // Process historical data
        // Group by topic and convert payloads to actual readings
        const processedData = {};

        data.forEach((reading) => {
          // Assuming each reading payload contains JSON with temperature, humidity, etc.
          // You'll need to adapt this based on your actual data format
          try {
            const decodedPayload = JSON.parse(reading.payload);

            if (!processedData[reading.topic]) {
              processedData[reading.topic] = [];
            }

            processedData[reading.topic].push({
              temperature: decodedPayload.temperature,
              humidity: decodedPayload.humidity,
              co2: decodedPayload.co2,
              timestamp: new Date(reading.received_at),
            });
          } catch (e) {
            console.error("Error processing historical data:", e);
          }
        });

        // Update charts with historical data
        if (processedData["AM300/OUTSIDE"] && processedData["AM300/INSIDE"]) {
          // Limit to last 50 readings
          const outsideData = processedData["AM300/OUTSIDE"].slice(-50);
          const insideData = processedData["AM300/INSIDE"].slice(-50);

          // Populate chart data
          for (
            let i = 0;
            i < Math.min(outsideData.length, insideData.length);
            i++
          ) {
            const time = outsideData[i].timestamp;

            tempChartData.push({
              time: time,
              inside: insideData[i].temperature,
              outside: outsideData[i].temperature,
            });

            humidityChartData.push({
              time: time,
              inside: insideData[i].humidity,
              outside: outsideData[i].humidity,
            });

            co2ChartData.push({
              time: time,
              inside: insideData[i].co2,
              outside: outsideData[i].co2,
            });
          }

          // Update the charts
          updateChart(tempChart, tempChartData, "Temperature (°C)");
          updateChart(humidityChart, humidityChartData, "Humidity (%)");
          updateChart(co2Chart, co2ChartData, "CO2 (ppm)");

          // Update UI with latest readings
          if (outsideData.length > 0) {
            latestReadings["AM300/OUTSIDE"] =
              outsideData[outsideData.length - 1];
          }

          if (insideData.length > 0) {
            latestReadings["AM300/INSIDE"] = insideData[insideData.length - 1];
          }

          updateUI();
        }
      })
      .catch((error) => {
        console.error("Error fetching historical data:", error);
      });
  }

  function formatTimestamp(timestamp) {
    return (
      timestamp.toLocaleTimeString() + " " + timestamp.toLocaleDateString()
    );
  }
}

// Start the dashboard when DOM is loaded
document.addEventListener("DOMContentLoaded", function () {
  // Check if Chart.js is already loaded
  if (typeof Chart !== "undefined") {
    console.log("Chart.js already loaded, initializing dashboard...");
    initDashboard();
  } else {
    console.log("Waiting for Chart.js to load...");
    // Listen for the custom event from our fallback loader
    document.addEventListener("chartsReady", function () {
      console.log("Chart.js now loaded, initializing dashboard...");
      initDashboard();
    });

    // Set a timeout in case the event never fires
    setTimeout(function () {
      if (typeof Chart === "undefined") {
        console.error("Chart.js failed to load after timeout");
        document.getElementById("temp-chart-error").classList.remove("d-none");
        document
          .getElementById("humidity-chart-error")
          .classList.remove("d-none");
        document.getElementById("co2-chart-error").classList.remove("d-none");
      } else if (!window.dashboardInitialized) {
        console.log(
          "Chart.js loaded but dashboard not initialized yet, initializing now...",
        );
        initDashboard();
      }
    }, 3000);
  }
});

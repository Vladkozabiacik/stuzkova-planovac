let userIP;
let ws;
const reconnectDelay = 5000;

function connectWebSocket() {
    ws = new WebSocket("ws://192.168.1.179:8080/ws");

    ws.onopen = function () {
        console.log("WebSocket connection established.");
        document.getElementById("wsStatus").innerText = "WebSocket Status: Connected";
        console.log("Status updated to Connected");
        document.addEventListener('mousemove', sendCursorPosition);
    };

    ws.onmessage = function (event) {
        handleMessage(event);
    };

    ws.onclose = function () {
        console.log("WebSocket connection closed. Attempting to reconnect...");
        document.getElementById("wsStatus").innerText = "WebSocket Status: Disconnected";
        console.log("Status updated to Disconnected");
        document.removeEventListener('mousemove', sendCursorPosition);
        setTimeout(connectWebSocket, reconnectDelay);
    };

    ws.onerror = function (error) {
        console.error("WebSocket error observed:", error);
    };
}

function handleMessage(event) {
    try {
        const data = JSON.parse(event.data);
        if (data.type === 'message') {
            appendMessage(data.text, data.user);
        } else if (data.type === 'cursor') {
            cursors[data.ip] = { x: data.x, y: data.y };
            renderCursors();
        } else if (data.type === 'disconnected') {
            delete cursors[data.ip];
            renderCursors();
        }
    } catch (error) {
        console.error("Error parsing message:", error);
    }
}

fetch("/get-ip")
    .then(response => response.json())
    .then(data => {
        userIP = data.ip;
    });

window.onload = function () {
    connectWebSocket();
};
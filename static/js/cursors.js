const cursors = {};

function sendCursorPosition(event) {
    const rect = document.body.getBoundingClientRect();
    let x = event.clientX - rect.left + window.scrollX;
    let y = event.clientY - rect.top + window.scrollY;

    x = Math.max(0, Math.min(window.innerWidth - 10, x));
    y = Math.max(0, Math.min(window.innerHeight - 10, y));

    ws.send(JSON.stringify({ type: 'cursor', x: x, y: y, ip: userIP }));
}

function renderCursors() {
    const cursorsDiv = document.getElementById("cursors");
    cursorsDiv.innerHTML = '';

    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;

    for (const [ip, pos] of Object.entries(cursors)) {
        if (ip === userIP) continue;

        let cursorPixel = document.getElementById(`cursor-${ip}`);

        if (!cursorPixel) {
            cursorPixel = document.createElement('div');
            cursorPixel.id = `cursor-${ip}`;
            cursorPixel.style.position = 'absolute';
            cursorPixel.style.width = '10px';
            cursorPixel.style.height = '10px';
            cursorPixel.style.borderRadius = '50%';
            cursorPixel.style.backgroundColor = 'red';
            cursorPixel.style.transition = 'left 0.1s ease, top 0.1s ease';
            cursorPixel.title = ip;
            cursorsDiv.appendChild(cursorPixel);
        }

        const boundedX = Math.max(0, Math.min(viewportWidth - 10, pos.x));
        const boundedY = Math.max(0, Math.min(viewportHeight - 10, pos.y));

        cursorPixel.style.left = `${boundedX}px`;
        cursorPixel.style.top = `${boundedY}px`;
    }
}

function updateCursors(ip, x, y) {
    cursors[ip] = { x, y };
    renderCursors();
}

document.addEventListener('mousemove', sendCursorPosition);
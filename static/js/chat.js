function appendMessage(message, user) {
    const chat = document.getElementById('chat');
    const messageDiv = document.createElement('div');
    messageDiv.classList.add('p-2', 'border-b');
    messageDiv.textContent = `${user}: ${message}`;
    chat.appendChild(messageDiv);
    chat.scrollTop = chat.scrollHeight;
}

function sendMessage() {
    const input = document.getElementById("message");
    const message = input.value;
    if (!message) return;

    if (!currentUser) {
        alert("You must be logged in to send messages.");
        return;
    }

    const messageObject = {
        type: 'message',
        text: message,
        user: currentUser
    };
    ws.send(JSON.stringify(messageObject));
    input.value = '';
    input.focus();
}

function toggleChat() {
    const chatContainer = document.getElementById('chatContainer');
    const chatButton = document.getElementById('chatButton');

    if (chatContainer.classList.contains('hidden')) {
        chatContainer.classList.remove('hidden');
        chatButton.textContent = 'Close Chat';
    } else {
        chatContainer.classList.add('hidden');
        chatButton.textContent = 'Chat';
    }
}

document.getElementById("message").addEventListener("keydown", function (event) {
    if (event.key === "Enter") {
        event.preventDefault();
        sendMessage();
    }
});

document.getElementById('chatButton').addEventListener('click', toggleChat);
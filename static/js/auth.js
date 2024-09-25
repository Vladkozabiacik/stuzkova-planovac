function showLoginForm() {
    document.getElementById('authTitle').innerText = 'Login';
    document.getElementById('authSubmit').innerText = 'Login';
    document.getElementById('switchToRegister').innerText = '✖';
    document.getElementById('authContainer').classList.remove('hidden');
}

function showRegisterForm() {
    document.getElementById('authTitle').innerText = 'Register';
    document.getElementById('authSubmit').innerText = 'Register';
    document.getElementById('switchToRegister').innerText = '✖';
    document.getElementById('authContainer').classList.remove('hidden');
}

function toggleAuthForm() {
    const authContainer = document.getElementById('authContainer');
    authContainer.classList.toggle('hidden');
}

document.getElementById('authSubmit').addEventListener('click', function () {
    const username = document.getElementById('authUsername').value;
    const password = document.getElementById('authPassword').value;
    const isLogin = document.getElementById('authTitle').innerText === 'Login';

    const url = isLogin ? '/login' : '/register';

    fetch(url, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ username, password })
    }).then(response => {
        if (response.ok) {
            currentUser = username;
            document.getElementById('authMessage').innerText = 'Success!';
            toggleAuthForm();
            checkLoginStatus();
        } else {
            return response.json().then(data => {
                document.getElementById('authMessage').innerText = data.message;
            });
        }
    }).catch(error => {
        console.error('Error:', error);
    });
});

function checkLoginStatus() {
    fetch('/session-status')
        .then(response => response.json())
        .then(data => {
            updateSessionStatus(data);
        })
        .catch(error => {
            console.error('Error checking login status:', error);
        });
}

function updateSessionStatus(data) {
    const sessionStatus = document.getElementById('sessionStatus');
    const logoutButton = document.getElementById('logoutButton');
    const loginButton = document.getElementById('loginButton');
    const registerButton = document.getElementById('registerButton');

    if (data.loggedIn) {
        sessionStatus.textContent = `Session Status: Logged in as ${data.username}`;
        currentUser = data.username;
        logoutButton.classList.remove('hidden');
        loginButton.classList.add('hidden');
        registerButton.classList.add('hidden');
    } else {
        sessionStatus.textContent = `Session Status: Not logged in`;
        currentUser = '';
        logoutButton.classList.add('hidden');
        loginButton.classList.remove('hidden');
        registerButton.classList.remove('hidden');
    }
}

// Fetch session status on page load
document.addEventListener("DOMContentLoaded", function () {
    fetch('/session-status')
        .then(response => response.json())
        .then(updateSessionStatus);
});

function logout() {
    fetch('/logout', { method: 'POST' })
        .then(response => {
            if (response.ok) {
                const logoutButton = document.getElementById('logoutButton');
                logoutButton.classList.add('hidden');
                updateSessionStatus({ loggedIn: false });
            } else {
                alert('Logout failed');
            }
        })
        .catch(error => {
            console.error('Error during logout:', error);
            alert('Logout error');
        });
}

// Call checkLoginStatus on page load
document.addEventListener('DOMContentLoaded', checkLoginStatus);
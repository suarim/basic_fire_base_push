document.addEventListener("DOMContentLoaded", async function () {
    function log(message) {
        const logDiv = document.getElementById('log');
        logDiv.innerHTML += message + '<br>';
        console.log(message);
    }

    async function initNotifications() {
        try {
            log('Initializing Firebase...');

            // Initialize Firebase
            const firebaseConfig = {
                apiKey: "",
                authDomain: "",
                projectId: "",
                storageBucket: "",
                messagingSenderId: "",
                appId: "",
                measurementId: ""
            };

            firebase.initializeApp(firebaseConfig);
            const messaging = firebase.messaging();

            // Request notification permission
            const permission = await Notification.requestPermission();
            log('Notification Permission: ' + permission);

            if (permission === 'granted') {
                // Register service worker
                const registration = await navigator.serviceWorker.register('sw.js');
                log('Service Worker Registered');

                // Get the device token
                const token = await messaging.getToken({
                    vapidKey: 'BATkzpqAykasP-nWVBfPxIFPbETTZHrt605mmx1EtsRtv-9pUNwVLmguKzTlgb3Gaj47fc_VH6gSyi5EIu9AzOw'
                });

                log('Device Token: ' + token);

                // Send this token to your backend to store for future notifications
                sendTokenToServer(token);

                // Listen for foreground notifications
                messaging.onMessage((payload) => {
                    console.log('Foreground Message:', payload);
                    const { title, body } = payload.notification;

                    new Notification(title, { body });
                });
            }
        } catch (error) {
            log('Error: ' + error.message);
            console.error(error);
        }
    }

    function sendTokenToServer(token) {
        log('Sending token to server...');
        // Here, make an API request to your backend to store the token.
    }

    document.getElementById("start-notifications").addEventListener("click", initNotifications);
});

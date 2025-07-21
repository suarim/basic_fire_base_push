importScripts("https://www.gstatic.com/firebasejs/11.3.0/firebase-app.js");
importScripts("https://www.gstatic.com/firebasejs/11.3.0/firebase-messaging.js");

// Firebase Configuration (Same as in index.html)
const firebaseConfig = {
    apiKey: "",
    authDomain: "",
    projectId: "",
    storageBucket: "",
    messagingSenderId: "",
    appId: "",
    measurementId: ""
};

// Initialize Firebase
firebase.initializeApp(firebaseConfig);
const messaging = firebase.messaging();

// Handle Background Notifications
messaging.onBackgroundMessage((payload) => {
    console.log('📩 Background notification:', payload);
    self.registration.showNotification(payload.notification.title, {
        body: payload.notification.body
    });
});

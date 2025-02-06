importScripts('https://www.gstatic.com/firebasejs/9.23.0/firebase-app-compat.js');
importScripts('https://www.gstatic.com/firebasejs/9.23.0/firebase-messaging-compat.js');

firebase.initializeApp({
    apiKey: ""*************************",",
    authDomain: "first-846ad.firebaseapp.com",
    projectId: "first-846ad",
    storageBucket: "first-846ad.firebasestorage.app",
    messagingSenderId: "661662103746",
    appId: "1:661662103746:web:f5fc8189f9703690c4a35b"
});

self.addEventListener('push', function(event) {
    console.log('Push received');
});
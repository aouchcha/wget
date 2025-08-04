import { renderLogin, renderProfile } from "./handlers.js"

// Application initialization
document.addEventListener('DOMContentLoaded', () => {
    const token = localStorage.getItem('JWT');
    token ? renderProfile() : renderLogin();
})

import { clearAndHideNotifications, decrypt, showErrorNotification } from "./core.mjs";

document.addEventListener("DOMContentLoaded", function() {
    const decryptSecretForm = document.getElementById("decryptSecretForm");
    if (!decryptSecretForm) {
        return;
    }

    decryptSecretForm
        .addEventListener("submit", async function(e) {
            e.preventDefault();

            clearAndHideNotifications(decryptSecretForm);

            const cipherText = decryptSecretForm.querySelector("input[name=cipherText]").value;
            const password = decryptSecretForm.querySelector("input[name=password]").value;

            try {
                const decryptedCipherText = await decrypt(cipherText, password);
                decryptSecretForm.querySelector("textarea[name=display]").value = decryptedCipherText;
            } catch (e) {
                showErrorNotification(decryptSecretForm, "Unable to decrypt secret. Have you entered the correct password?");
            }
        });
});

import { decrypt } from "./crypto.mjs";

document.addEventListener("DOMContentLoaded", function() {
    const decryptSecretForm = document.getElementById("decryptSecretForm");
    if (!decryptSecretForm) {
        return;
    }

    decryptSecretForm
        .querySelector("button")
        .addEventListener("click", async function(e) {
            e.preventDefault();

            const errorContainer = decryptSecretForm.querySelector("#decryptSecretError");
            errorContainer.innerHTML = '';

            const cipherText = decryptSecretForm.querySelector("input[name=cipherText]").value;
            const password = decryptSecretForm.querySelector("input[name=password]").value;

            try {
                const decryptedCipherText = await decrypt(cipherText, password);
                decryptSecretForm.querySelector("textarea[name=display]").value = decryptedCipherText;
            } catch (e) {
                errorContainer.innerHTML = "Unable to decrypt secret. Have you entered the correct password?";
            }
        });
});

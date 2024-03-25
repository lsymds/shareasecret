import {
	clearAndHideNotifications,
	decrypt,
	showErrorNotification,
} from "./core.mjs";

document.addEventListener("DOMContentLoaded", function () {
	const decryptSecretForm = document.getElementById("decryptSecretForm");
	if (!decryptSecretForm) {
		return;
	}

	decryptSecretForm.addEventListener("submit", async function (e) {
		e.preventDefault();

		clearAndHideNotifications(decryptSecretForm);

		const submitButton = decryptSecretForm.querySelector("button");
		const cipherTextInput = decryptSecretForm.querySelector(
			"input[name=cipherText]"
		);
		const passwordInput = decryptSecretForm.querySelector(
			"input[name=password]"
		);
		const decryptedCipherTextInput = decryptSecretForm.querySelector(
			"textarea[name=display]"
		);

		try {
			const decryptedCipherText = await decrypt(
				cipherTextInput.value,
				passwordInput.value
			);
			decryptedCipherTextInput.value = decryptedCipherText;

			decryptedCipherTextInput.removeAttribute("disabled");
			decryptedCipherTextInput.focus();

			submitButton.setAttribute("disabled", "true");
			passwordInput.setAttribute("disabled", "true");
		} catch (e) {
			console.error(e);
			showErrorNotification(
				decryptSecretForm,
				"Unable to decrypt secret. Have you entered the correct password?"
			);
		}
	});
});

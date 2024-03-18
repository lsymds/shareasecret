import { Controller } from "https://unpkg.com/@hotwired/stimulus/dist/stimulus.js";
import { encrypt } from "./crypto.mjs";

class CreateSecretController extends Controller {
    static targets = [
        'encryptedSecret',
        'plaintextSecret',
        'password',
        'ttl'
    ];

    async encryptAndSubmit(event) {
        event.preventDefault();

        const response = await encrypt(this.plaintextSecretTarget.value, this.passwordTarget.value);
        this.encryptedSecretTarget.value = response;
    }
}

window.Stimulus.register('createSecret', CreateSecretController);

import { Controller } from "https://unpkg.com/@hotwired/stimulus/dist/stimulus.js";

class CreateSecretController extends Controller {
    static targets = ['encryptedSecret', 'plaintextSecret', 'password', 'ttl'];

    encryptAndSubmit(event) {
        event.preventDefault();
    }
}

window.Stimulus.register('createSecret', CreateSecretController);

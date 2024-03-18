import { encrypt } from "./crypto.mjs";

export default function() {
    return {
        plaintextSecret: '',
        password: '',
        encryptedSecret: '',

        async encryptAndSubmit(e) {
            e.preventDefault();

            this.encryptedSecret = await encrypt(this.plaintextSecret, this.password);
            await this.$nextTick();
            window.htmx.trigger(this.$el, 'createSecret');
        }
    }
}

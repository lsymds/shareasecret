document.addEventListener("DOMContentLoaded", function () {
	document.querySelectorAll(".j-button--copy").forEach(function (b) {
		b.addEventListener("click", async function () {
			const clipboardPermission = await navigator.permissions.query({
				name: "clipboard-write",
			});
			if (clipboardPermission.state !== "granted") {
				console.error(
					".j-button--copy hook: clipboard write permission not granted"
				);
				return;
			}

			const targetAttribute = b.attributes.getNamedItem("data-target");
			if (!targetAttribute) {
				console.error(
					".j-button--copy hook: data-target attribute is required"
				);
				return;
			}

			const target = document.querySelector(
				`input[name=${targetAttribute.value}]`
			);
			if (!target) {
				console.error(".j-button--copy hook: data-target provided not found");
				return;
			}

			target.select();
			target.setSelectionRange(0, 99999);

			navigator.clipboard.writeText(target.value);
		});
	});
});

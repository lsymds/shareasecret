document.addEventListener("DOMContentLoaded", function () {
	document.querySelectorAll(".j-button--copy").forEach(function (b) {
		b.addEventListener("click", async function () {
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

			target.removeAttribute("disabled");
			target.select();
			document.execCommand("copy");
			target.setAttribute("disabled", true);
		});
	});
});

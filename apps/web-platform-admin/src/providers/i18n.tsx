import i18n from "i18next";
import type { PropsWithChildren } from "react";
import { I18nextProvider, initReactI18next } from "react-i18next";

const resources = {
	en: {
		translation: {
			dashboard: "Dashboard",
			users: "Users",
			schemaCategories: "Schema Categories",
			notFoundTitle: "Page not found",
			backToDashboard: "Go to dashboard",
		},
	},
};

let initialized = false;

function ensureI18n() {
	if (initialized) return;
	i18n.use(initReactI18next).init({
		resources,
		lng: "en",
		fallbackLng: "en",
		interpolation: { escapeValue: false },
	});
	initialized = true;
}

export function I18nProvider({ children }: PropsWithChildren) {
	ensureI18n();
	return <I18nextProvider i18n={i18n}>{children}</I18nextProvider>;
}

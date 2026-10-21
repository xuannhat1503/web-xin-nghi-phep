import { getContext, setContext } from 'svelte';
import { en } from './en.ts';
import { vi, type MessageKey } from './vi.ts';
import { serverVi } from './server-vi.ts';
import type { Catalog, LeaveType } from '#lib/admin-types.ts';

export type Lang = 'vi' | 'en';
export const LANGS: Lang[] = ['vi', 'en'];
export const DEFAULT_LANG: Lang = 'vi';
export const LANG_COOKIE = 'lang';
export type { MessageKey };

export type Params = Record<string, string | number>;

/** A message to show: a dictionary key with params, or a list joined by spaces. */
export type Msg = MessageKey | { key: MessageKey; params?: Params } | Msg[];

export function isLang(v: unknown): v is Lang {
	return v === 'vi' || v === 'en';
}

const dicts: Record<Lang, Record<MessageKey, string>> = { vi, en };

export function translate(lang: Lang, key: MessageKey, params?: Params): string {
	const text: string = dicts[lang][key] ?? dicts.en[key] ?? key;
	if (!params) return text;
	return text.replace(/\{(\w+)\}/g, (m, k: string) => (k in params ? String(params[k]) : m));
}

export function hasKey(key: string): key is MessageKey {
	return Object.hasOwn(vi, key);
}

export function renderMsg(lang: Lang, m: Msg): string {
	if (Array.isArray(m))
		return m
			.map((x) => renderMsg(lang, x))
			.filter(Boolean)
			.join(' ');
	if (typeof m === 'string') return translate(lang, m);
	return translate(lang, m.key, m.params);
}

/** Translates an English message produced by the Go API (exact sentence match); unknown text is returned as is. */
export function serverText(lang: Lang, message: string | undefined | null): string {
	if (!message) return '';
	if (lang === 'en') return message;
	return serverVi[message] ?? message;
}

export interface I18n {
	readonly lang: Lang;
	t: (key: MessageKey, params?: Params) => string;
	msg: (m: Msg | null | undefined) => string;
	server: (message: string | null | undefined) => string;
	/** All leave types in display order (inactive ones too, for filters and history). */
	readonly leaveTypes: LeaveType[];
	/** Name of a leave type in the current language (falls back to the code). */
	leaveType: (code: string | null | undefined) => string;
	/** Label of a reason in the current language; `stored` is used when the code is unknown. */
	reason: (code: string | null | undefined, stored?: string | null) => string;
}

const KEY = Symbol('i18n');

/** Called once in the root layout. The getters must read reactive state so components update on switch. */
export function provideI18n(getLang: () => Lang, getCatalog: () => Catalog): I18n {
	const i18n: I18n = {
		get lang() {
			return getLang();
		},
		t: (key, params) => translate(getLang(), key, params),
		msg: (m) => (m ? renderMsg(getLang(), m) : ''),
		server: (message) => serverText(getLang(), message),
		get leaveTypes() {
			return getCatalog().leave_types;
		},
		leaveType: (code) => {
			if (!code) return '';
			const t = getCatalog().leave_types.find((x) => x.code === code);
			return t ? (getLang() === 'en' ? t.name_en : t.name_vi) : code;
		},
		reason: (code, stored) => {
			if (!code) return '';
			const r = getCatalog().reasons.find((x) => x.code === code);
			if (!r) return stored ?? code;
			return getLang() === 'en' ? r.label_en || r.label : r.label;
		}
	};
	setContext(KEY, i18n);
	return i18n;
}

export function useI18n(): I18n {
	return getContext<I18n>(KEY);
}

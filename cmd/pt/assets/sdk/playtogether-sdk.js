// ../../../../../../../../../Users/yanghsuanming/Sites/play-together/node_modules/zod/v4/core/util.js
function e(e) {
	let t = Object.values(e).filter((e) => typeof e == "number");
	return Object.entries(e).filter(([e, n]) => t.indexOf(+e) === -1).map(([e, t]) => t);
}
function t(e, t) {
	return typeof t == "bigint" ? t.toString() : t;
}
var n = class {
	constructor(e) {
		this._getter = e, this._value = void 0;
	}
	get value() {
		let e = this._getter;
		return e !== void 0 && (this._value = e(), this._getter = void 0), this._value;
	}
};
function r(e) {
	return new n(e);
}
function i(e) {
	return e == null;
}
function a(e) {
	let t = +!!e.startsWith("^"), n = e.endsWith("$") ? e.length - 1 : e.length;
	return e.slice(t, n);
}
function o(e, t, n) {
	Object.defineProperty(e, t, {
		value: n,
		writable: !0,
		enumerable: !0,
		configurable: !0
	});
}
var s = "captureStackTrace" in Error ? Error.captureStackTrace : (...e) => {};
function c(e) {
	return typeof e == "object" && !!e && !Array.isArray(e);
}
function l(e) {
	if (c(e) === !1) return !1;
	let t = e.constructor;
	if (t === void 0 || typeof t != "function") return !0;
	let n = t.prototype;
	return c(n) !== !1 && Object.prototype.hasOwnProperty.call(n, "isPrototypeOf") !== !1;
}
function u(e) {
	return l(e) ? { ...e } : Array.isArray(e) ? [...e] : e instanceof Map ? new Map(e) : e instanceof Set ? new Set(e) : e;
}
var d = /* @__PURE__*/ new Set([
	"string",
	"number",
	"symbol"
]);
function f(e) {
	return e.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
function p(e, t, n) {
	let r = new e._zod.constr(t ?? e._zod.def);
	return (!t || n?.parent) && (r._zod.parent = e), r;
}
function m(e) {
	let t = e;
	if (!t) return {};
	if (typeof t == "string") return { error: () => t };
	if (t?.message !== void 0) {
		if (t?.error !== void 0) throw Error("Cannot specify both `message` and `error` params");
		t.error = t.message;
	}
	return delete t.message, typeof t.error == "string" ? {
		...t,
		error: () => t.error
	} : t;
}
function h(e) {
	return Object.keys(e).filter((t) => e[t]._zod.optin !== void 0 && e[t]._zod.optout === "optional");
}
var g = {
	safeint: [-(2 ** 53 - 1), 2 ** 53 - 1],
	int32: [-2147483648, 2147483647],
	uint32: [0, 4294967295],
	float32: [-34028234663852886e22, 34028234663852886e22],
	float64: [-Number.MAX_VALUE, Number.MAX_VALUE]
};
function _(e, t = 0) {
	if (e.aborted === !0) return !0;
	for (let n = t; n < e.issues.length; n++) if (e.issues[n]?.continue !== !0) return !0;
	return !1;
}
function ee(e, t = 0) {
	if (e.aborted === !0) return !0;
	for (let n = t; n < e.issues.length; n++) if (e.issues[n]?.continue === !1) return !0;
	return !1;
}
function v(e, t) {
	return t.map((t) => {
		var n;
		return (n = t).path ?? (n.path = []), t.path.unshift(e), t;
	});
}
function y(e) {
	return typeof e == "string" ? e : e?.message;
}
function te(e) {
	let t = [];
	for (let n of e ?? []) {
		let e = n._zod.def;
		if (!/_(length|size)$|_equals$/.test(e.check)) break;
		let r = e.maximum ?? e.length ?? e.size;
		r >= 0 && t.push([r, n]);
	}
	return t;
}
function ne(e, t, n) {
	var r;
	for (let i = t; i < e.length; i++) (r = e[i]).schema ?? (r.schema = n);
}
function b(e, t, n) {
	var r;
	let i = e.inst?._zod?.traits;
	i?.has("$ZodType") && (i.has("$ZodCheck") ? (r = e).schema ?? (r.schema = e.inst) : e.schema = e.inst);
	let a = e.schema === e.inst ? void 0 : e.schema?._zod.def?.error, o = e.message ? e.message : y(e.inst?._zod.def?.error?.(e)) ?? y(a?.(e)) ?? y(t?.error?.(e)) ?? y(n.customError?.(e)) ?? y(n.localeError?.(e)) ?? "Invalid input", s = {};
	for (let t of Object.keys(e)) t !== "inst" && t !== "schema" && t !== "continue" && t !== "input" && t !== "__proto__" && (s[t] = e[t]);
	return s.path ?? (s.path = []), s.message = o, t?.reportInput && (s.input = e.input), s;
}
var re = /[\uD800-\uDBFF]/;
function ie(e) {
	let t = e.length;
	if (!re.test(e)) return t;
	let n = t;
	for (let r = 0; r < t - 1; r++) (e.charCodeAt(r) & 64512) == 55296 && (e.charCodeAt(r + 1) & 64512) == 56320 && (n--, r++);
	return n;
}
function ae(e) {
	return Array.isArray(e) ? "array" : typeof e == "string" ? "string" : "unknown";
}
function oe(...e) {
	let [t, n, r] = e;
	return typeof t == "string" ? {
		message: t,
		code: "custom",
		input: n,
		inst: r
	} : { ...t };
}
function se(e, t) {
	for (let n in t) {
		let r = Object.getOwnPropertyDescriptor(t, n);
		r.get ? Object.defineProperty(e, n, {
			...r,
			enumerable: !1
		}) : le(e, n, r.value);
	}
}
function x(e, t, n, r = !0) {
	return Object.defineProperty(e, t, {
		configurable: !0,
		writable: !0,
		enumerable: r,
		value: n
	}), n;
}
function ce(e, t, n) {
	return x(e, t, n, !1);
}
function le(e, t, n) {
	Object.defineProperty(e, t, {
		configurable: !0,
		get() {
			return this == null ? n : x(this, t, n.bind(this));
		},
		set(e) {
			x(this, t, e);
		}
	});
}
function ue(e, t) {
	let n = Object.getPrototypeOf(e);
	return t in n ? void 0 : n;
}
var de, S = !1, fe = {
	configurable: !0,
	get() {
		S = !0;
	}
};
function C(e, t, n) {
	let r = Object.getPrototypeOf(e._zod);
	t in r && de !== e._zod ? de = void 0 : (de = e._zod, Object.defineProperty(r, t, {
		configurable: !0,
		get() {
			Object.defineProperty(this, t, fe);
			let e = S;
			S = !1;
			try {
				let r = n(this);
				return S ? delete this[t] : Object.defineProperty(this, t, {
					configurable: !0,
					writable: !0,
					value: r
				}), S = S || e, r;
			} catch (n) {
				throw delete this[t], S = S || e, n;
			}
		},
		set(e) {
			Object.defineProperty(this, t, {
				configurable: !0,
				writable: !0,
				value: e
			});
		}
	}));
}
function pe(e, t, n, r) {
	let i = ue(e, t);
	i && Object.defineProperty(i, t, {
		configurable: !0,
		get() {
			let e = {
				configurable: !0,
				writable: !0,
				enumerable: r,
				value: void 0
			};
			return Object.defineProperty(this, t, e), e.value = n(this), Object.defineProperty(this, t, e), e.value;
		},
		set(e) {
			Object.defineProperty(this, t, {
				configurable: !0,
				writable: !0,
				enumerable: r,
				value: e
			});
		}
	});
}
// ../../../../../../../../../Users/yanghsuanming/Sites/play-together/node_modules/zod/v4/core/core.js
var me, he = {
	value: void 0,
	enumerable: !1
}, ge = "captureStackTrace" in Error ? Error : null;
function _e(e) {
	let t = ge;
	if (t) {
		let n = t.stackTraceLimit;
		if (typeof n == "number") {
			try {
				t.stackTraceLimit = 0;
			} catch {
				return ge = null, new e();
			}
			try {
				return new e();
			} finally {
				t.stackTraceLimit = n;
			}
		}
	}
	return new e();
}
/* @__NO_SIDE_EFFECTS__ */
function w(e, t, n, r) {
	let i = {};
	function a(e) {
		this.def = e, this.constr = d, this.traits = /* @__PURE__ */ new Set();
	}
	a.prototype = i;
	let o = n, s = o && /* @__PURE__ */ new WeakSet();
	function c(n, r) {
		if (!n._zod) {
			he.value = new a(r);
			try {
				Object.defineProperty(n, "_zod", he);
			} finally {
				he.value = void 0;
			}
		} else if (n._zod.traits.has(e)) return;
		if (n._zod.traits.add(e), t(n, r), s) {
			let e = Object.getPrototypeOf(n), t = n._zod.constr.prototype, r = e;
			for (; r && r !== t;) r = Object.getPrototypeOf(r);
			let i = r ?? e;
			s.has(i) || (s.add(i), se(i, o));
		}
		let i = d.prototype;
		for (let e in i) Object.prototype.hasOwnProperty.call(i, e) && (e in n || (n[e] = i[e].bind(n)));
	}
	let l = r?.Parent ?? Object;
	class u extends l {}
	Object.defineProperty(u, "name", { value: e });
	function d(e) {
		let t = r?.Parent ? _e(u) : this;
		c(t, e);
		let n = t._zod.deferred;
		if (n) {
			for (let e of n) e();
			t._zod.deferred = void 0;
		}
		let i = globalThis.__zod_globalConfig?.postProcessor;
		return i && i(t), t;
	}
	return Object.defineProperty(d, "init", { value: c }), Object.defineProperty(d, Symbol.hasInstance, { value: (t) => r?.Parent && t instanceof r.Parent ? !0 : t?._zod?.traits?.has(e) }), Object.defineProperty(d, "name", { value: e }), d;
}
var T = class extends Error {
	constructor() {
		super("Encountered Promise during synchronous parse. Use .parseAsync() instead.");
	}
};
(me = globalThis).__zod_globalConfig ?? (me.__zod_globalConfig = {});
var E = globalThis.__zod_globalConfig;
function D(e) {
	return e && Object.assign(E, e), E;
}
// ../../../../../../../../../Users/yanghsuanming/Sites/play-together/node_modules/zod/v4/core/errors.js
function ve() {
	let e = this._zod;
	return e.message ?? (e.message = JSON.stringify(e.def, t, 2)), e.message;
}
function ye(e) {
	this._zod.message = e;
}
var be = {
	get: ve,
	set: ye,
	enumerable: !0,
	configurable: !0
}, xe = {
	value: void 0,
	enumerable: !1
}, Se = /* @__PURE__ */ new WeakSet([Object.prototype, Error.prototype]), O = /* @__PURE__ */ w("$ZodError", (e, t) => {
	e.name = "$ZodError", xe.value = t, Object.defineProperty(e, "issues", xe), xe.value = void 0, Object.defineProperty(e, "message", be);
	let n = Object.getPrototypeOf(e);
	Se.has(n) || (Se.add(n), Object.defineProperty(n, "toString", {
		configurable: !0,
		enumerable: !1,
		get() {
			let e = () => this.message;
			return Object.defineProperty(this, "toString", {
				value: e,
				configurable: !0,
				writable: !0
			}), e;
		},
		set(e) {
			Object.defineProperty(this, "toString", {
				value: e,
				configurable: !0,
				writable: !0
			});
		}
	}));
}, void 0, { Parent: Error }), Ce = /* @__PURE__*/ ((e) => {
	let t = (n, r, i, a) => {
		let o = i ? {
			...i,
			async: !1
		} : { async: !1 }, c = n._zod.run({
			value: r,
			issues: []
		}, o);
		if (c instanceof Promise) throw new T();
		if (c.issues.length) {
			let n = new ((a?.Err) ?? e)(c.issues.map((e) => b(e, o, D())));
			throw s(n, a?.callee ?? t), n;
		}
		return c.value;
	};
	return t;
})(O), we = /* @__PURE__*/ ((e) => {
	let t = async (n, r, i, a) => {
		let o = i ? {
			...i,
			async: !0
		} : { async: !0 }, c = n._zod.run({
			value: r,
			issues: []
		}, o);
		if (c instanceof Promise && (c = await c), c.issues.length) {
			let n = new ((a?.Err) ?? e)(c.issues.map((e) => b(e, o, D())));
			throw s(n, a?.callee ?? t), n;
		}
		return c.value;
	};
	return t;
})(O), Te = /* @__PURE__*/ ((e) => (t, n, r) => {
	let i = r ? {
		...r,
		async: !1
	} : { async: !1 }, a = t._zod.run({
		value: n,
		issues: []
	}, i);
	if (a instanceof Promise) throw new T();
	return a.issues.length ? Ee(e, a.issues, i) : {
		success: !0,
		data: a.value
	};
})(O);
function Ee(e, t, n) {
	let r;
	return {
		success: !1,
		get error() {
			return r || (r = new e(t.map((e) => b(e, n, D()))), t = void 0, n = void 0), r;
		},
		set error(e) {
			r = e, t = void 0, n = void 0;
		}
	};
}
var De = /* @__PURE__*/ ((e) => async (t, n, r) => {
	let i = r ? {
		...r,
		async: !0
	} : { async: !0 }, a = t._zod.run({
		value: n,
		issues: []
	}, i);
	return a instanceof Promise && (a = await a), a.issues.length ? Ee(e, a.issues, i) : {
		success: !0,
		data: a.value
	};
})(O), Oe = /^[\s\S]{0,}$/, ke = /^-?\d+(?:\.\d+)?$/, Ae = /^(?:true|false)$/i, k = /*@__PURE__*/ w("$ZodCheck", (e, t) => {
	var n;
	e._zod ?? (e._zod = {}), e._zod.def = t, (n = e._zod).onattach ?? (n.onattach = []);
}), je = (e) => {
	let t = e.value;
	return !i(t) && t.length !== void 0;
}, A = {
	number: "number",
	bigint: "bigint",
	object: "date"
}, Me = /*@__PURE__*/ w("$ZodCheckLessThan", (e, t) => {
	k.init(e, t);
	let n = A[typeof t.value];
	e._zod.check = (r) => {
		(t.inclusive ? r.value <= t.value : r.value < t.value) || r.issues.push({
			origin: A[typeof r.value] ?? n,
			code: "too_big",
			maximum: typeof t.value == "object" ? t.value.getTime() : t.value,
			input: r.value,
			inclusive: t.inclusive,
			inst: e,
			continue: !t.abort
		});
	};
}), Ne = /*@__PURE__*/ w("$ZodCheckGreaterThan", (e, t) => {
	k.init(e, t);
	let n = A[typeof t.value];
	e._zod.check = (r) => {
		(t.inclusive ? r.value >= t.value : r.value > t.value) || r.issues.push({
			origin: A[typeof r.value] ?? n,
			code: "too_small",
			minimum: typeof t.value == "object" ? t.value.getTime() : t.value,
			input: r.value,
			inclusive: t.inclusive,
			inst: e,
			continue: !t.abort
		});
	};
}), Pe = /*@__PURE__*/ w("$ZodCheckNumberFormat", (e, t) => {
	k.init(e, t), t.format = t.format || "float64";
	let n = t.format?.includes("int"), r = n ? "int" : "number", [i, a] = g[t.format];
	e._zod.check = (o) => {
		let s = o.value;
		if (n) {
			if (!Number.isInteger(s)) {
				o.issues.push({
					expected: r,
					format: t.format,
					code: "invalid_type",
					continue: !1,
					input: s,
					inst: e
				});
				return;
			}
			if (!Number.isSafeInteger(s)) {
				s > 0 ? o.issues.push({
					input: s,
					code: "too_big",
					maximum: 2 ** 53 - 1,
					note: "Integers must be within the safe integer range.",
					inst: e,
					origin: r,
					inclusive: !0,
					continue: !t.abort
				}) : o.issues.push({
					input: s,
					code: "too_small",
					minimum: -(2 ** 53 - 1),
					note: "Integers must be within the safe integer range.",
					inst: e,
					origin: r,
					inclusive: !0,
					continue: !t.abort
				});
				return;
			}
		}
		s < i && o.issues.push({
			origin: "number",
			input: s,
			code: "too_small",
			minimum: i,
			inclusive: !0,
			inst: e,
			continue: !t.abort
		}), s > a && o.issues.push({
			origin: "number",
			input: s,
			code: "too_big",
			maximum: a,
			inclusive: !0,
			inst: e,
			continue: !t.abort
		});
	};
}), Fe = /*@__PURE__*/ w("$ZodCheckMaxLength", (e, t) => {
	var n;
	k.init(e, t), (n = e._zod.def).when ?? (n.when = je), e._zod.check = (n) => {
		let r = n.value, i = r.length;
		if ((typeof r == "string" && i > t.maximum ? ie(r) : i) <= t.maximum) return;
		let a = ae(r);
		n.issues.push({
			origin: a,
			code: "too_big",
			maximum: t.maximum,
			inclusive: !0,
			input: r,
			inst: e,
			continue: !t.abort
		});
	};
}), Ie = /*@__PURE__*/ w("$ZodCheckMinLength", (e, t) => {
	var n;
	k.init(e, t), (n = e._zod.def).when ?? (n.when = je), e._zod.check = (n) => {
		let r = n.value, i = r.length;
		if ((typeof r == "string" && i >= t.minimum && i < t.minimum * 2 ? ie(r) : i) >= t.minimum) return;
		let a = ae(r);
		n.issues.push({
			origin: a,
			code: "too_small",
			minimum: t.minimum,
			inclusive: !0,
			input: r,
			inst: e,
			continue: !t.abort
		});
	};
}), Le = /*@__PURE__*/ w("$ZodCheckStringFormat", (e, t) => {
	var n, r;
	k.init(e, t), t.pattern ? (n = e._zod).check ?? (n.check = (n) => {
		t.pattern.lastIndex = 0, t.pattern.test(n.value) || n.issues.push({
			origin: "string",
			code: "invalid_format",
			format: t.format,
			input: n.value,
			...t.pattern ? { pattern: t.pattern.toString() } : {},
			inst: e,
			continue: !t.abort
		});
	}) : (r = e._zod).check ?? (r.check = () => {});
}), Re = /*@__PURE__*/ w("$ZodCheckRegex", (e, t) => {
	Le.init(e, t), e._zod.check = (n) => {
		t.pattern.lastIndex = 0, t.pattern.test(n.value) || n.issues.push({
			origin: "string",
			code: "invalid_format",
			format: "regex",
			input: n.value,
			pattern: t.pattern.toString(),
			inst: e,
			continue: !t.abort
		});
	};
}), ze = {
	major: 4,
	minor: 6,
	patch: 7
}, j = /*@__PURE__*/ w("$ZodType", (e, t) => {
	var n;
	e ?? (e = {}), e._zod.def = t, e._zod.bag = e._zod.bag || {}, e._zod.version = ze;
	let r = e._zod.def.checks, i = e._zod.traits.has("$ZodCheck") ? [e, ...r ?? []] : r?.length ? [...r] : [];
	for (let t of i) for (let n of t._zod.onattach) n(e);
	if (i.length === 0) (n = e._zod).deferred ?? (n.deferred = []), e._zod.deferred?.push(() => {
		e._zod.run = e._zod.parse;
	});
	else {
		let t = (t, n, r) => {
			if (t.memo) return t;
			let i = _(t), a;
			for (let o of n) {
				if (o._zod.def.when) {
					if (ee(t) || !o._zod.def.when(t)) continue;
				} else if (i) continue;
				let n = t.issues.length, s = o._zod.check(t);
				if (s instanceof Promise && r?.async === !1) throw new T();
				if (a || s instanceof Promise) a = (a ?? Promise.resolve()).then(async () => {
					await s, t.issues.length !== n && (ne(t.issues, n, e), i || (i = _(t, n)));
				});
				else {
					if (t.issues.length === n) continue;
					ne(t.issues, n, e), i || (i = _(t, n));
				}
			}
			return a ? a.then(() => t) : t;
		}, n = (n, r, a) => {
			if (_(n)) return n.aborted = !0, n;
			let o = t(r, i, a);
			if (o instanceof Promise) {
				if (a.async === !1) throw new T();
				return o.then((t) => e._zod.parse(t, a));
			}
			return e._zod.parse(o, a);
		};
		e._zod.run = (r, a) => {
			if (a.skipChecks) return e._zod.parse(r, a);
			if (a.direction === "backward") {
				let t = e._zod.parse({
					value: r.value,
					issues: []
				}, {
					...a,
					skipChecks: !0
				});
				return t instanceof Promise ? t.then((e) => n(e, r, a)) : n(t, r, a);
			}
			let o = e._zod.parse(r, a);
			if (o instanceof Promise) {
				if (a.async === !1) throw new T();
				return o.then((e) => t(e, i, a));
			}
			return t(o, i, a);
		};
	}
}, {
	get "~standard"() {
		return ce(this, "~standard", He(this));
	},
	set "~standard"(e) {
		x(this, "~standard", e);
	}
}), Be = (e, t) => e.issues.length ? { issues: e.issues.map((e) => b(e, t, D())) } : { value: e.value };
async function Ve(e, t) {
	let n = { async: !0 };
	return Be(await e._zod.run({
		value: t,
		issues: []
	}, n), n);
}
function He(e) {
	return {
		validate: (t) => {
			let n = { async: !1 };
			try {
				let r = e._zod.run({
					value: t,
					issues: []
				}, n);
				if (!(r instanceof Promise)) return Be(r, n);
			} catch {}
			return Ve(e, t);
		},
		vendor: "zod",
		version: 1
	};
}
var Ue = /*@__PURE__*/ w("$ZodString", (e, t) => {
	j.init(e, t), e._zod.pattern = t.pattern ?? Oe, e._zod.parse = (n, r) => {
		if (t.coerce) try {
			n.value = String(n.value);
		} catch {}
		return typeof n.value == "string" || n.issues.push({
			expected: "string",
			code: "invalid_type",
			input: n.value,
			inst: e
		}), n;
	};
}), We = /*@__PURE__*/ w("$ZodNumber", (e, t) => {
	j.init(e, t), e._zod.pattern = ke, e._zod.parse = (n, r) => {
		if (t.coerce) try {
			n.value = Number(n.value);
		} catch {}
		let i = n.value;
		if (typeof i == "number" && !Number.isNaN(i) && Number.isFinite(i)) return n;
		let a = typeof i == "number" ? Number.isNaN(i) ? "NaN" : Number.isFinite(i) ? void 0 : String(i) : void 0;
		return n.issues.push({
			expected: "number",
			code: "invalid_type",
			input: i,
			inst: e,
			...a ? { received: a } : {}
		}), n;
	};
}), Ge = /*@__PURE__*/ w("$ZodNumberFormat", (e, t) => {
	Pe.init(e, t), We.init(e, t);
}), Ke = /*@__PURE__*/ w("$ZodBoolean", (e, t) => {
	j.init(e, t), e._zod.pattern = Ae, e._zod.parse = (n, r) => {
		if (t.coerce) try {
			n.value = !!n.value;
		} catch {}
		let i = n.value;
		return typeof i == "boolean" || n.issues.push({
			expected: "boolean",
			code: "invalid_type",
			input: i,
			inst: e
		}), n;
	};
}), qe = /*@__PURE__*/ w("$ZodUnknown", (e, t) => {
	j.init(e, t), e._zod.parse = (e) => e;
});
function Je(e, t) {
	let n = te(e._zod.def.checks);
	if (!n.length) return;
	let r = Math.min(...n.map((e) => e[0])), i = e._zod.parse;
	e._zod.parse = (n, a) => {
		let o = t(n.value);
		if (o > r && !a?.skipChecks && a?.direction !== "backward") {
			for (let [t, r] of te(e._zod.def.checks)) if (o > t && r._zod.def.when(n)) {
				let t = n.issues.length;
				return r._zod.check(n), n.issues[t].continue = !1, ne(n.issues, t, e), n;
			}
		}
		return i(n, a);
	};
}
var Ye = (e) => Array.isArray(e) ? e.length : -1;
function Xe(e, t, n) {
	e.issues.length && t.issues.push(...v(n, e.issues)), t.value[n] = e.value;
}
var Ze = /*@__PURE__*/ w("$ZodArray", (e, t) => {
	j.init(e, t);
	let n = E.memoizer;
	n?.attach(e), e._zod.parse = (r, i) => {
		let a = r.value;
		if (!Array.isArray(a)) return r.issues.push({
			expected: "array",
			code: "invalid_type",
			input: a,
			inst: e
		}), r;
		r.value = n ? n.alloc(e, r, Array(a.length), i) : Array(a.length);
		let o = [], s = i?.abortEarly;
		for (let e = 0; e < a.length; e++) {
			let n = a[e], c = t.element._zod.run({
				value: n,
				issues: []
			}, i);
			if (c instanceof Promise) o.push(c.then((t) => Xe(t, r, e)));
			else if (Xe(c, r, e), s && c.issues.length !== 0 && _(c)) break;
		}
		return o.length ? Promise.all(o).then(() => r) : r;
	}, Je(e, Ye);
});
function M(e, t, n, r, i, a) {
	let o = n in r, s = a === "optional";
	if (o || !s || i !== "optional") {
		if (e.issues.length) {
			if (i !== void 0 && s && !o) return;
			t.issues.push(...v(n, e.issues));
		}
		!o && i === void 0 ? e.issues.length || t.issues.push({
			code: "invalid_type",
			expected: "nonoptional",
			input: void 0,
			path: [n]
		}) : e.value === void 0 ? (o || i === "defaulted" && !s) && (t.value[n] = void 0) : t.value[n] = e.value;
	}
}
var Qe = [];
function $e(e) {
	let t = Object.keys(e.shape), n = Object.getOwnPropertySymbols(e.shape), r = n.length ? n : Qe, i = r.length ? [...t, ...r] : t;
	for (let t of i) if (!e.shape?.[t]?._zod?.traits?.has("$ZodType")) throw Error(`Invalid element at key "${String(t)}": expected a Zod schema`);
	let a = h(e.shape);
	return {
		...e,
		allKeys: i,
		symbolKeys: r,
		keySet: new Set(t),
		numKeys: t.length,
		optionalKeys: new Set(a)
	};
}
function et(e, t, n, r, i, a, o) {
	let s = [], c = i.keySet, l = i.catchall._zod, u = l.def.type, d = l.optin, f = l.optout, p = 0;
	for (let i in t) {
		if (o && n.issues.length !== p) {
			if (_(n, p)) break;
			p = n.issues.length;
		}
		if (c.has(i)) continue;
		if (i === "__proto__") {
			u === "never" && s.push(i);
			continue;
		}
		if (u === "never") {
			s.push(i);
			continue;
		}
		let a = l.run({
			value: t[i],
			issues: []
		}, r);
		a instanceof Promise ? e.push(a.then((e) => M(e, n, i, t, d, f))) : M(a, n, i, t, d, f);
	}
	return s.length && n.issues.push({
		code: "unrecognized_keys",
		keys: s,
		input: t,
		inst: a,
		continue: !0
	}), e.length ? Promise.all(e).then(() => n) : n;
}
var tt = /*@__PURE__*/ w("$ZodObject", (e, t) => {
	j.init(e, t);
	let n = Object.getOwnPropertyDescriptor(t, "shape"), i = n?.get ? n.get.raw : t.shape ?? {};
	if (i) {
		let e = () => {
			let n = { ...i };
			return Object.defineProperty(t, "shape", { value: n }), e.raw = n, n;
		};
		e.raw = i, Object.defineProperty(t, "shape", { get: e });
	}
	let a = r(() => $e(t));
	C(e, "propValues", (e) => {
		let t = e.def.shape, n = {};
		for (let e in t) {
			let r = t[e]._zod;
			if (r.values) {
				Object.prototype.hasOwnProperty.call(n, e) || o(n, e, /* @__PURE__ */ new Set());
				for (let t of r.values) n[e].add(t);
				r.optin !== void 0 && n[e].add(void 0);
			}
		}
		return n;
	});
	let s = c, l = t.catchall, u, d = E.memoizer;
	d?.attach(e), e._zod.parse = (t, n) => {
		u ?? (u = a.value);
		let r = t.value;
		if (!s(r)) return t.issues.push({
			expected: "object",
			code: "invalid_type",
			input: r,
			inst: e
		}), t;
		t.value = d ? d.alloc(e, t, {}, n) : {};
		let i = [], o = u.shape, c = n?.abortEarly, f = t.issues.length;
		for (let e of u.allKeys) {
			if (c && t.issues.length !== f) {
				if (_(t, f)) break;
				f = t.issues.length;
			}
			if (e === "__proto__") continue;
			let a = o[e], s = a._zod.optin, l = a._zod.optout, u = a._zod.run({
				value: r[e],
				issues: []
			}, n);
			u instanceof Promise ? i.push(u.then((n) => M(n, t, e, r, s, l))) : M(u, t, e, r, s, l);
		}
		return l ? et(i, r, t, n, a.value, e, c === !0) : i.length ? Promise.all(i).then(() => t) : t;
	};
});
function nt(e, t, n, r) {
	for (let n of e) if (n.issues.length === 0) return t.value = n.value, t;
	let i = e.filter((e) => !_(e));
	return i.length === 1 ? (t.value = i[0].value, i[0]) : (t.issues.push({
		code: "invalid_union",
		input: t.value,
		inst: n,
		errors: e.map((e) => e.issues.map((e) => b(e, r, D())))
	}), t);
}
var rt = /*@__PURE__*/ w("$ZodUnion", (e, t) => {
	j.init(e, t), C(e, "optin", (e) => e.def.options.some((e) => e._zod.optin === "defaulted") ? "defaulted" : e.def.options.some((e) => e._zod.optin !== void 0) ? "optional" : void 0), C(e, "optout", (e) => e.def.options.some((e) => e._zod.optout === "optional") ? "optional" : void 0), C(e, "values", (e) => {
		if (e.def.options.every((e) => e._zod.values)) return new Set(e.def.options.flatMap((e) => Array.from(e._zod.values)));
	}), C(e, "pattern", (e) => {
		if (e.def.options.every((e) => e._zod.pattern)) {
			let t = e.def.options.map((e) => e._zod.pattern);
			return RegExp(`^(${t.map((e) => a(e.source)).join("|")})$`);
		}
	});
	let n = t.options.length === 1 ? t.options[0]._zod.run : null;
	e._zod.parse = (r, i) => {
		if (n) return n(r, i);
		let a = !1, o = [];
		for (let e of t.options) {
			let t = e._zod.run({
				value: r.value,
				issues: []
			}, i);
			if (t instanceof Promise) o.push(t), a = !0;
			else {
				if (t.issues.length === 0) return t;
				o.push(t);
			}
		}
		return a ? Promise.all(o).then((t) => nt(t, r, e, i)) : nt(o, r, e, i);
	};
}), it = /*@__PURE__*/ w("$ZodRecord", (e, t) => {
	j.init(e, t);
	let n = E.memoizer;
	n?.attach(e), e._zod.parse = (r, i) => {
		let a = r.value;
		if (!l(a)) return r.issues.push({
			expected: "record",
			code: "invalid_type",
			input: a,
			inst: e
		}), r;
		let o = [], s = t.keyType._zod.values;
		if (s && !t.partial) {
			r.value = n ? n.alloc(e, r, {}, i) : {};
			let c = /* @__PURE__ */ new Set();
			for (let n of s) if (typeof n == "string" || typeof n == "number" || typeof n == "symbol") {
				if (c.add(typeof n == "number" ? n.toString() : n), n === "__proto__") continue;
				let s = t.keyType._zod.run({
					value: n,
					issues: []
				}, i);
				if (s instanceof Promise) throw Error("Async schemas not supported in object keys currently");
				if (s.issues.length) {
					r.issues.push({
						code: "invalid_key",
						origin: "record",
						issues: s.issues.map((e) => b(e, i, D())),
						input: n,
						path: [n],
						inst: e
					});
					continue;
				}
				let l = s.value;
				if (l === "__proto__") continue;
				let u = t.valueType._zod.run({
					value: a[n],
					issues: []
				}, i);
				u instanceof Promise ? o.push(u.then((e) => {
					e.issues.length && r.issues.push(...v(n, e.issues)), r.value[l] = e.value;
				})) : (u.issues.length && r.issues.push(...v(n, u.issues)), r.value[l] = u.value);
			}
			let l;
			for (let e in a) if (!c.has(e)) {
				if (t.mode === "loose") {
					if (e === "__proto__") continue;
					r.value[e] = a[e];
				} else l = l ?? [], l.push(e);
			}
			l && l.length > 0 && r.issues.push({
				code: "unrecognized_keys",
				input: a,
				inst: e,
				keys: l,
				continue: !0
			});
		} else {
			r.value = n ? n.alloc(e, r, {}, i) : {};
			let c;
			for (let n of Reflect.ownKeys(a)) {
				if (n === "__proto__" || !Object.prototype.propertyIsEnumerable.call(a, n)) continue;
				let l = t.keyType._zod.run({
					value: n,
					issues: []
				}, i);
				if (l instanceof Promise) throw Error("Async schemas not supported in object keys currently");
				if (typeof n == "string" && ke.test(n) && l.issues.length) {
					let e = t.keyType._zod.run({
						value: Number(n),
						issues: []
					}, i);
					if (e instanceof Promise) throw Error("Async schemas not supported in object keys currently");
					e.issues.length === 0 && (l = e);
				}
				if (l.issues.length) {
					t.mode === "loose" ? r.value[n] = a[n] : s ? (c = c ?? [], c.push(n)) : r.issues.push({
						code: "invalid_key",
						origin: "record",
						issues: l.issues.map((e) => b(e, i, D())),
						input: n,
						path: [n],
						inst: e
					});
					continue;
				}
				let u = l.value;
				if (u === "__proto__") continue;
				let d = t.valueType._zod.run({
					value: a[n],
					issues: []
				}, i);
				d instanceof Promise ? o.push(d.then((e) => {
					e.issues.length && r.issues.push(...v(n, e.issues)), r.value[u] = e.value;
				})) : (d.issues.length && r.issues.push(...v(n, d.issues)), r.value[u] = d.value);
			}
			c && c.length > 0 && r.issues.push({
				code: "unrecognized_keys",
				input: a,
				inst: e,
				keys: c,
				continue: !0
			});
		}
		return o.length ? Promise.all(o).then(() => r) : r;
	};
}), at = /*@__PURE__*/ w("$ZodEnum", (t, n) => {
	j.init(t, n);
	let r = e(n.entries), i = new Set(r);
	t._zod.values = i, C(t, "pattern", (t) => {
		let n = e(t.def.entries).filter((e) => d.has(typeof e));
		return RegExp(n.length ? `^(${n.map((e) => f(e.toString())).join("|")})$` : "^[^\\s\\S]$");
	}), t._zod.parse = (e, n) => {
		let a = e.value;
		return i.has(a) || e.issues.push({
			code: "invalid_value",
			values: r,
			input: a,
			inst: t
		}), e;
	};
}), ot = /*@__PURE__*/ w("$ZodLiteral", (e, t) => {
	j.init(e, t);
	let n = new Set(t.values);
	e._zod.values = n, C(e, "pattern", (e) => {
		let t = e.def.values;
		return RegExp(t.length ? `^(${t.map((e) => typeof e == "string" ? f(e) : e ? f(e.toString()) : String(e)).join("|")})$` : "^[^\\s\\S]$");
	}), e._zod.parse = (r, i) => {
		let a = r.value;
		return n.has(a) || r.issues.push({
			code: "invalid_value",
			values: t.values,
			input: a,
			inst: e
		}), r;
	};
});
function st(e, t) {
	return e.value = t.issues.length ? void 0 : t.value, e;
}
var ct = /*@__PURE__*/ w("$ZodOptional", (e, t) => {
	j.init(e, t), C(e, "optin", (e) => e.def.innerType._zod.optin === "defaulted" ? "defaulted" : "optional"), e._zod.optout = "optional", C(e, "values", (e) => {
		let t = e.def.innerType._zod.values;
		return t ? /* @__PURE__ */ new Set([...t, void 0]) : void 0;
	}), C(e, "pattern", (e) => {
		let t = e.def.innerType._zod.pattern;
		return t ? RegExp(`^(${a(t.source)})?$`) : void 0;
	}), e._zod.parse = (e, n) => {
		if (e.value === void 0) {
			if (t.innerType._zod.optin !== "defaulted") return e;
			let r = t.innerType._zod.run({
				value: e.value,
				issues: []
			}, n);
			return r instanceof Promise ? r.then((t) => st(e, t)) : st(e, r);
		}
		return t.innerType._zod.run(e, n);
	};
}), lt = /*@__PURE__*/ w("$ZodNullable", (e, t) => {
	j.init(e, t), C(e, "optin", (e) => e.def.innerType._zod.optin), C(e, "optout", (e) => e.def.innerType._zod.optout), C(e, "pattern", (e) => {
		let t = e.def.innerType._zod.pattern;
		return t ? RegExp(`^(${a(t.source)}|null)$`) : void 0;
	}), C(e, "values", (e) => e.def.innerType._zod.values ? /* @__PURE__ */ new Set([...e.def.innerType._zod.values, null]) : void 0), e._zod.parse = (e, n) => e.value === null ? e : t.innerType._zod.run(e, n);
}), ut = /*@__PURE__*/ w("$ZodDefault", (e, t) => {
	j.init(e, t), e._zod.optin = "defaulted", C(e, "values", (e) => e.def.innerType._zod.values), e._zod.parse = (e, n) => {
		if (n.direction === "backward") return t.innerType._zod.run(e, n);
		if (e.value === void 0) return e.value = t.defaultValue, e;
		let r = t.innerType._zod.run(e, n);
		return r instanceof Promise ? r.then((e) => dt(e, t)) : dt(r, t);
	};
});
function dt(e, t) {
	return e.value === void 0 && (e.value = t.defaultValue), e;
}
var ft = /*@__PURE__*/ w("$ZodCustom", (e, t) => {
	k.init(e, t), j.init(e, t), e._zod.parse = (e, t) => e, e._zod.check = (n) => {
		let r = n.value, i = t.fn(r);
		if (i instanceof Promise) return i.then((t) => pt(t, n, r, e));
		pt(i, n, r, e);
	};
});
function pt(e, t, n, r) {
	if (!e) {
		let e = {
			code: "custom",
			input: n,
			inst: r,
			path: [...r._zod.def.path ?? []],
			continue: !r._zod.def.abort
		};
		r._zod.def.params && (e.params = r._zod.def.params), t.issues.push(oe(e));
	}
}
// ../../../../../../../../../Users/yanghsuanming/Sites/play-together/node_modules/zod/v4/core/api.js
function mt(e) {
	return e.checks && (e.checks = [...e.checks]), e;
}
// @__NO_SIDE_EFFECTS__
function ht(e, t) {
	return new e(mt({
		type: "string",
		...m(t)
	}));
}
// @__NO_SIDE_EFFECTS__
function gt(e, t) {
	return new e(mt({
		type: "number",
		checks: [],
		...m(t)
	}));
}
// @__NO_SIDE_EFFECTS__
function _t(e, t) {
	return new e({
		type: "number",
		check: "number_format",
		abort: !1,
		format: "safeint",
		...m(t)
	});
}
// @__NO_SIDE_EFFECTS__
function vt(e, t) {
	return new e({
		type: "boolean",
		...m(t)
	});
}
// @__NO_SIDE_EFFECTS__
function yt(e) {
	return new e({ type: "unknown" });
}
// @__NO_SIDE_EFFECTS__
function N(e, t) {
	return new Me({
		check: "less_than",
		...m(t),
		value: e,
		inclusive: !0
	});
}
// @__NO_SIDE_EFFECTS__
function bt(e, t) {
	return new Ne({
		check: "greater_than",
		...m(t),
		value: e,
		inclusive: !1
	});
}
// @__NO_SIDE_EFFECTS__
function P(e, t) {
	return new Ne({
		check: "greater_than",
		...m(t),
		value: e,
		inclusive: !0
	});
}
// @__NO_SIDE_EFFECTS__
function F(e) {
	return /* @__PURE__ */ bt(0, e);
}
// @__NO_SIDE_EFFECTS__
function I(e, t) {
	return new Fe({
		check: "max_length",
		...m(t),
		maximum: e
	});
}
// @__NO_SIDE_EFFECTS__
function L(e, t) {
	return new Ie({
		check: "min_length",
		...m(t),
		minimum: e
	});
}
// @__NO_SIDE_EFFECTS__
function R(e, t) {
	return new Re({
		check: "string_format",
		format: "regex",
		...m(t),
		pattern: e
	});
}
// @__NO_SIDE_EFFECTS__
function xt(e, t, n) {
	return new e({
		type: "custom",
		check: "custom",
		fn: t,
		...m(n)
	});
}
// ../../../../../../../../../Users/yanghsuanming/Sites/play-together/node_modules/zod/v4/mini/schemas.js
var z = /*@__PURE__*/ w("ZodMiniType", (e, t) => {
	if (!e._zod) throw Error("Uninitialized schema in ZodMiniType.");
	j.init(e, t), e.def = t, e.type = t.type;
}, {
	get with() {
		return this.check;
	},
	set with(e) {
		x(this, "with", e);
	},
	parse(e, t) {
		return Ce(this, e, t, { callee: this.parse });
	},
	parseAsync(e, t) {
		return we(this, e, t, { callee: this.parseAsync });
	},
	safeParse(e, t) {
		return Te(this, e, t);
	},
	safeParseAsync(e, t) {
		return De(this, e, t);
	},
	check(...e) {
		let t = this.def;
		return this.clone({
			...t,
			checks: [...t.checks ?? [], ...e.map((e) => typeof e == "function" ? { _zod: {
				check: e,
				def: { check: "custom" },
				onattach: []
			} } : e)]
		}, { parent: !0 });
	},
	clone(e, t) {
		return p(this, e, t);
	},
	brand() {
		return this;
	},
	register(e, t) {
		return e.add(this, t), this;
	},
	apply(e, ...t) {
		return t.length === 0 ? e(this) : e(this, ...t);
	}
}), St = /*@__PURE__*/ w("ZodMiniString", (e, t) => {
	Ue.init(e, t), z.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function B(e) {
	return /* @__PURE__ */ ht(St, e);
}
var Ct = /*@__PURE__*/ w("ZodMiniNumber", (e, t) => {
	We.init(e, t), z.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function wt(e) {
	return /* @__PURE__ */ gt(Ct, e);
}
var Tt = /*@__PURE__*/ w("ZodMiniNumberFormat", (e, t) => {
	Ge.init(e, t), Ct.init(e, t);
});
function V(e) {
	return /* @__PURE__ */ _t(Tt, e);
}
var Et = /*@__PURE__*/ w("ZodMiniBoolean", (e, t) => {
	Ke.init(e, t), z.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function H(e) {
	return /* @__PURE__ */ vt(Et, e);
}
var Dt = /*@__PURE__*/ w("ZodMiniUnknown", (e, t) => {
	qe.init(e, t), z.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Ot() {
	return /* @__PURE__ */ yt(Dt);
}
var kt = /*@__PURE__*/ w("ZodMiniArray", (e, t) => {
	Ze.init(e, t), z.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function U(e, t) {
	return new kt({
		type: "array",
		element: e,
		...m(t)
	});
}
var At = /*@__PURE__*/ w("ZodMiniObject", (e, t) => {
	tt.init(e, t), z.init(e, t), pe(e, "shape", (e) => e._zod.def.shape, !1);
});
// @__NO_SIDE_EFFECTS__
function W(e, t) {
	return new At({
		type: "object",
		shape: e ?? {},
		...m(t)
	});
}
var jt = /*@__PURE__*/ w("ZodMiniUnion", (e, t) => {
	rt.init(e, t), z.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Mt(e, t) {
	return new jt({
		type: "union",
		options: e,
		...m(t)
	});
}
var Nt = /*@__PURE__*/ w("ZodMiniRecord", (e, t) => {
	it.init(e, t), z.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Pt(e, t, n) {
	return !t || !t._zod ? new Nt({
		type: "record",
		keyType: /* @__PURE__ */ B(),
		valueType: e,
		...m(t)
	}) : new Nt({
		type: "record",
		keyType: e,
		valueType: t,
		...m(n)
	});
}
var Ft = /*@__PURE__*/ w("ZodMiniEnum", (e, t) => {
	at.init(e, t), z.init(e, t), e.options = [...e._zod.values];
});
// @__NO_SIDE_EFFECTS__
function G(e, t) {
	return new Ft({
		type: "enum",
		entries: Array.isArray(e) ? Object.fromEntries(e.map((e) => [e, e])) : e,
		...m(t)
	});
}
var It = /*@__PURE__*/ w("ZodMiniLiteral", (e, t) => {
	ot.init(e, t), z.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function K(e, t) {
	return new It({
		type: "literal",
		values: Array.isArray(e) ? e : [e],
		...m(t)
	});
}
var Lt = /*@__PURE__*/ w("ZodMiniOptional", (e, t) => {
	ct.init(e, t), z.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function q(e) {
	return new Lt({
		type: "optional",
		innerType: e
	});
}
var Rt = /*@__PURE__*/ w("ZodMiniNullable", (e, t) => {
	lt.init(e, t), z.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function J(e) {
	return new Rt({
		type: "nullable",
		innerType: e
	});
}
var zt = /*@__PURE__*/ w("ZodMiniDefault", (e, t) => {
	ut.init(e, t), z.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Bt(e, t) {
	return new zt({
		type: "default",
		innerType: e,
		get defaultValue() {
			return typeof t == "function" ? t() : u(t);
		}
	});
}
var Vt = /*@__PURE__*/ w("ZodMiniCustom", (e, t) => {
	ft.init(e, t), z.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Y(e, t = {}) {
	return /* @__PURE__ */ xt(Vt, e, t);
}
var X = (/* @__PURE__ */ B()).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(128)), Ht = (/* @__PURE__ */ B()).check(/* @__PURE__ */ I(64)), Ut = (/* @__PURE__ */ B()).check(/* @__PURE__ */ I(256), /* @__PURE__ */ R(/^https?:\/\/d-[a-z0-9]{16}\.[a-z0-9.-]+(:[0-9]{1,5})?$/)), Wt = (/* @__PURE__ */ B()).check(/* @__PURE__ */ R(/^pur_[A-Za-z0-9_-]{22}$/)), Gt = (/* @__PURE__ */ B()).check(/* @__PURE__ */ R(/^pool_[A-Za-z0-9_-]{22}$/)), Z = (e, t) => (/* @__PURE__ */ wt()).check(/* @__PURE__ */ P(e), /* @__PURE__ */ N(t)), Kt = /* @__PURE__ */ W({
	id: X,
	name: Ht
}), qt = /* @__PURE__ */ G([
	"open",
	"host_only",
	"turn",
	"allowlist"
]), Jt = /* @__PURE__ */ G(["ptt", "hands_free"]), Yt = /* @__PURE__ */ G(["self"]), Xt = /* @__PURE__ */ G([
	"game",
	"user",
	"denied",
	"not_authorized",
	"mic_denied",
	"unsupported",
	"background",
	"timeout",
	"busy",
	"closed"
]), Zt = Jt, Qt = /* @__PURE__ */ G(["self"]), $t = /* @__PURE__ */ G(["zh-TW"]), en = Xt, tn = 5e3, nn = /^[a-z][a-z0-9_]{0,15}$/, rn = (/* @__PURE__ */ B()).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(60), /* @__PURE__ */ R(/^[^\u0000-\u001f\u007f]+$/)), an = /* @__PURE__ */ Mt([/* @__PURE__ */ W({
	type: /* @__PURE__ */ K("integer"),
	min: V().check(/* @__PURE__ */ P(0), /* @__PURE__ */ N(9999)),
	max: V().check(/* @__PURE__ */ P(0), /* @__PURE__ */ N(9999))
}), /* @__PURE__ */ W({
	type: /* @__PURE__ */ K("enum"),
	values: (/* @__PURE__ */ U((/* @__PURE__ */ B()).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(20), /* @__PURE__ */ R(/^[^\s{}\u0000-\u001f\u007f]+$/)))).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(200))
})]);
function on(e) {
	return [...e.matchAll(/\{([^{}]*)\}/g)].map((e) => e[1] ?? "");
}
function sn(e) {
	let t = 0;
	for (let n of e.phrases) {
		let r = 1;
		for (let t of on(n)) {
			let n = e.slots?.[t];
			if (!n || !nn.test(t)) return -1;
			let i = n.type === "integer" ? n.max - n.min + 1 : n.values.length;
			if (i < 1) return -1;
			if (r *= i, r > 5e3) return 5001;
		}
		if (/[{}]/.test(n.replace(/\{[^{}]*\}/g, ""))) return -1;
		if (t += r, t > 5e3) return 5001;
	}
	return t;
}
var cn = (/* @__PURE__ */ W({
	phrases: (/* @__PURE__ */ U(rn)).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(50)),
	slots: /* @__PURE__ */ q(/* @__PURE__ */ Pt((/* @__PURE__ */ B()).check(/* @__PURE__ */ R(nn)), an)),
	strict: /* @__PURE__ */ H()
})).check(/* @__PURE__ */ Y((e) => Object.keys(e.slots ?? {}).length <= 8, "欄位最多 8 個"), /* @__PURE__ */ Y((e) => new Set(e.phrases).size === e.phrases.length, "句型不得重複"), /* @__PURE__ */ Y((e) => {
	let t = sn(e);
	return t >= 1 && t <= 5e3;
}, `句型用到的欄位要有定義，展開後最多 ${tn} 句`));
(/* @__PURE__ */ B()).check(/* @__PURE__ */ I(32)), Z(0, 1), V().check(/* @__PURE__ */ P(0), /* @__PURE__ */ N(1e3)), Z(0, 86400);
var ln = /* @__PURE__ */ W({
	top: Z(0, 1e4),
	right: Z(0, 1e4),
	bottom: Z(0, 1e4),
	left: Z(0, 1e4)
}), un = {
	"sdk.init": /* @__PURE__ */ W({
		protocolVersion: /* @__PURE__ */ K(1),
		launchCode: (/* @__PURE__ */ B()).check(/* @__PURE__ */ L(16), /* @__PURE__ */ I(256)),
		player: /* @__PURE__ */ W({
			name: Ht,
			locale: (/* @__PURE__ */ B()).check(/* @__PURE__ */ I(16))
		}),
		room: /* @__PURE__ */ W({ id: X }),
		server: /* @__PURE__ */ q(/* @__PURE__ */ W({ url: Ut }))
	}),
	"app.pause": /* @__PURE__ */ W({}),
	"app.resume": /* @__PURE__ */ W({}),
	"app.close": /* @__PURE__ */ W({}),
	"audio.mute": /* @__PURE__ */ W({ muted: /* @__PURE__ */ H() }),
	"room.members": /* @__PURE__ */ W({ members: (/* @__PURE__ */ U(Kt)).check(/* @__PURE__ */ I(64)) }),
	"ui.insets": ln,
	"voice.state": /* @__PURE__ */ W({
		connected: /* @__PURE__ */ H(),
		canSpeak: /* @__PURE__ */ H(),
		mode: qt,
		group: /* @__PURE__ */ q(/* @__PURE__ */ J((/* @__PURE__ */ B()).check(/* @__PURE__ */ R(/^[a-z][a-z0-9_-]{0,15}$/))))
	}),
	"voice.speaking": /* @__PURE__ */ W({
		playerId: X,
		level: Z(0, 1)
	}),
	"purchase.result": /* @__PURE__ */ W({
		purchaseId: Wt,
		status: /* @__PURE__ */ G([
			"completed",
			"cancelled",
			"failed"
		])
	}),
	"pool.result": /* @__PURE__ */ W({
		poolId: Gt,
		status: /* @__PURE__ */ G([
			"staked",
			"declined",
			"failed"
		])
	}),
	"pitch.state": /* @__PURE__ */ W({
		state: /* @__PURE__ */ G(["started", "stopped"]),
		reason: /* @__PURE__ */ q(Xt)
	}),
	"voice.pitch": /* @__PURE__ */ W({
		playerId: X,
		sessionId: X,
		seq: V().check(/* @__PURE__ */ P(0)),
		t: Z(0, 0xe8d4a51000),
		serverTimeMs: Z(0, 0x5af3107a4000),
		voiced: /* @__PURE__ */ H(),
		hz: /* @__PURE__ */ J(Z(20, 5e3)),
		note: /* @__PURE__ */ J(Z(0, 127)),
		cents: /* @__PURE__ */ J(Z(-50, 50)),
		confidence: Z(0, 1)
	}),
	"speech.state": /* @__PURE__ */ W({
		state: /* @__PURE__ */ G(["started", "stopped"]),
		reason: /* @__PURE__ */ q(en)
	}),
	"voice.speech": /* @__PURE__ */ W({
		playerId: X,
		sessionId: X,
		seq: V().check(/* @__PURE__ */ P(0)),
		serverTimeMs: Z(0, 0x5af3107a4000),
		final: /* @__PURE__ */ H(),
		alternatives: (/* @__PURE__ */ U(/* @__PURE__ */ W({
			text: (/* @__PURE__ */ B()).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(100)),
			confidence: /* @__PURE__ */ J(Z(0, 1)),
			phrase: /* @__PURE__ */ q(rn),
			slots: /* @__PURE__ */ q(/* @__PURE__ */ Pt((/* @__PURE__ */ B()).check(/* @__PURE__ */ R(nn)), /* @__PURE__ */ Mt([V(), (/* @__PURE__ */ B()).check(/* @__PURE__ */ I(20))])))
		}))).check(/* @__PURE__ */ I(5))
	})
}, dn = /* @__PURE__ */ W({
	pt: /* @__PURE__ */ K("playtogether"),
	v: /* @__PURE__ */ K(1),
	type: (/* @__PURE__ */ B()).check(/* @__PURE__ */ I(64)),
	id: (/* @__PURE__ */ B()).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(64)),
	instance: /* @__PURE__ */ q((/* @__PURE__ */ B()).check(/* @__PURE__ */ I(64))),
	nonce: /* @__PURE__ */ q((/* @__PURE__ */ B()).check(/* @__PURE__ */ I(128))),
	payload: /* @__PURE__ */ Ot()
});
function fn(e) {
	try {
		return new TextEncoder().encode(JSON.stringify(e)).length;
	} catch {
		return Infinity;
	}
}
function pn(e) {
	if (fn(e) > 8192) return {
		ok: !1,
		error: "too_large"
	};
	let t = dn.safeParse(e);
	return t.success ? {
		ok: !0,
		value: t.data
	} : {
		ok: !1,
		error: "bad_envelope"
	};
}
function mn(e) {
	let t = pn(e);
	if (!t.ok) return t;
	let n = un[t.value.type];
	if (!n) return {
		ok: !1,
		error: "unknown_type"
	};
	let r = n.safeParse(t.value.payload);
	return r.success ? {
		ok: !0,
		value: {
			...t.value,
			payload: r.data
		}
	} : {
		ok: !1,
		error: "bad_payload"
	};
}
var hn = /* @__PURE__ */ W({
	id: (/* @__PURE__ */ B()).check(/* @__PURE__ */ R(/^[a-z][a-z0-9_-]{0,15}$/)),
	members: (/* @__PURE__ */ U(X)).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(64))
});
(/* @__PURE__ */ W({
	roomId: X,
	gameInstanceId: X,
	sequence: V().check(/* @__PURE__ */ P(0)),
	mode: qt,
	allowedSpeakerIds: /* @__PURE__ */ Bt((/* @__PURE__ */ U(X)).check(/* @__PURE__ */ I(64)), []),
	groups: /* @__PURE__ */ q((/* @__PURE__ */ U(hn)).check(/* @__PURE__ */ I(8))),
	effectiveAtServerMs: V().check(/* @__PURE__ */ F()),
	expiresAtServerMs: V().check(/* @__PURE__ */ F())
})).check(/* @__PURE__ */ Y((e) => e.expiresAtServerMs > e.effectiveAtServerMs, "expiresAtServerMs 必須晚於 effectiveAtServerMs"), /* @__PURE__ */ Y((e) => e.mode !== "turn" || e.allowedSpeakerIds.length === 1, "turn 模式必須剛好一位發言者"), /* @__PURE__ */ Y((e) => e.mode !== "allowlist" || e.allowedSpeakerIds.length >= 1 || (e.groups?.length ?? 0) > 0, "allowlist 模式至少要一位發言者"), /* @__PURE__ */ Y((e) => e.mode !== "open" && e.mode !== "host_only" || e.allowedSpeakerIds.length === 0, "open / host_only 模式不帶發言者名單"), /* @__PURE__ */ Y((e) => new Set(e.allowedSpeakerIds).size === e.allowedSpeakerIds.length, "發言者名單不得重複"), /* @__PURE__ */ Y((e) => new Set((e.groups ?? []).map((e) => e.id)).size === (e.groups ?? []).length, "分組 id 不得重複"), /* @__PURE__ */ Y((e) => {
	let t = (e.groups ?? []).flatMap((e) => e.members);
	return new Set(t).size === t.length;
}, "一個人最多在一組")), (/* @__PURE__ */ W({
	roomId: X,
	gameInstanceId: X,
	epoch: V().check(/* @__PURE__ */ P(0)),
	sequence: V().check(/* @__PURE__ */ P(0)),
	allowedPlayerIds: (/* @__PURE__ */ U(X)).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(64)),
	mode: Jt,
	audience: Yt,
	effectiveAtServerMs: V().check(/* @__PURE__ */ F()),
	expiresAtServerMs: V().check(/* @__PURE__ */ F())
})).check(/* @__PURE__ */ Y((e) => e.expiresAtServerMs > e.effectiveAtServerMs, "expiresAtServerMs 必須晚於 effectiveAtServerMs"), /* @__PURE__ */ Y((e) => new Set(e.allowedPlayerIds).size === e.allowedPlayerIds.length, "名單不得重複")), (/* @__PURE__ */ W({
	roomId: X,
	gameInstanceId: X,
	epoch: V().check(/* @__PURE__ */ P(0)),
	sequence: V().check(/* @__PURE__ */ P(0)),
	allowedPlayerIds: (/* @__PURE__ */ U(X)).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(64)),
	mode: Zt,
	audience: Qt,
	language: $t,
	vocabulary: /* @__PURE__ */ q(cn),
	effectiveAtServerMs: V().check(/* @__PURE__ */ F()),
	expiresAtServerMs: V().check(/* @__PURE__ */ F())
})).check(/* @__PURE__ */ Y((e) => e.expiresAtServerMs > e.effectiveAtServerMs, "expiresAtServerMs 必須晚於 effectiveAtServerMs"), /* @__PURE__ */ Y((e) => new Set(e.allowedPlayerIds).size === e.allowedPlayerIds.length, "名單不得重複")), (/* @__PURE__ */ B()).check(/* @__PURE__ */ L(16), /* @__PURE__ */ I(256)), V().check(/* @__PURE__ */ P(0)), (/* @__PURE__ */ U((/* @__PURE__ */ B()).check(/* @__PURE__ */ I(64)))).check(/* @__PURE__ */ I(500));
// \0@oxc-project+runtime@0.156.0/helpers/esm/typeof.js
function Q(e) {
	"@babel/helpers - typeof";
	return Q = typeof Symbol == "function" && typeof Symbol.iterator == "symbol" ? function(e) {
		return typeof e;
	} : function(e) {
		return e && typeof Symbol == "function" && e.constructor === Symbol && e !== Symbol.prototype ? "symbol" : typeof e;
	}, Q(e);
}
// \0@oxc-project+runtime@0.156.0/helpers/esm/toPrimitive.js
function gn(e, t) {
	if (Q(e) != "object" || !e) return e;
	var n = e[Symbol.toPrimitive];
	if (n !== void 0) {
		var r = n.call(e, t || "default");
		if (Q(r) != "object") return r;
		throw TypeError("@@toPrimitive must return a primitive value.");
	}
	return (t === "string" ? String : Number)(e);
}
// \0@oxc-project+runtime@0.156.0/helpers/esm/toPropertyKey.js
function _n(e) {
	var t = gn(e, "string");
	return Q(t) == "symbol" ? t : t + "";
}
// \0@oxc-project+runtime@0.156.0/helpers/esm/defineProperty.js
function vn(e, t, n) {
	return (t = _n(t)) in e ? Object.defineProperty(e, t, {
		value: n,
		enumerable: !0,
		configurable: !0,
		writable: !0
	}) : e[t] = n, e;
}
// src/index.ts
var yn = "0.1.0";
function bn(e = window) {
	return {
		embedded: e.parent !== e,
		parent: e.parent,
		href: e.location?.href,
		fetch: (t) => e.fetch(t, { cache: "no-store" }),
		post: (t, n) => e.parent.postMessage(t, n),
		listen(t) {
			let n = (e) => t({
				origin: e.origin,
				source: e.source,
				data: e.data
			});
			return e.addEventListener("message", n), () => e.removeEventListener("message", n);
		}
	};
}
var xn = 15e4, $ = class {
	constructor() {
		vn(this, "map", /* @__PURE__ */ new Map());
	}
	on(e, t) {
		let n = this.map.get(e);
		return n || this.map.set(e, n = /* @__PURE__ */ new Set()), n.add(t), () => n.delete(t);
	}
	emit(e, t) {
		for (let n of this.map.get(e) ?? []) try {
			n(t);
		} catch (e) {
			console.error("[playtogether] listener error", e);
		}
	}
};
function Sn(e, t, n) {
	let r = n - t, i = r / 1e3 % 6, a = {
		playerId: "mock-player",
		sessionId: `mock-${t}`,
		seq: e,
		t: r,
		serverTimeMs: n
	}, o = (e) => ({
		...a,
		voiced: !1,
		hz: null,
		note: null,
		cents: null,
		confidence: e
	});
	if (i >= 2 && i < 2.6) return o(.1);
	if (i >= 5) return o(.35);
	let s = i < 2 ? 220 * 2 ** (i / 2) : 440 * 2 ** (Math.sin(i * 2 * Math.PI * 5) * 15 / 1200), c = 69 + 12 * Math.log2(s / 440), l = Math.round(c);
	return {
		...a,
		voiced: !0,
		hz: s,
		note: l,
		cents: Math.max(-50, Math.min(50, (c - l) * 100)),
		confidence: .92
	};
}
var Cn = class extends Error {
	constructor() {
		super("無法和玩聚平台完成握手（逾時）"), this.name = "HandshakeTimeoutError";
	}
};
function wn() {
	let e = /* @__PURE__ */ new Uint8Array(16);
	return crypto.getRandomValues(e), Array.from(e, (e) => e.toString(16).padStart(2, "0")).join("");
}
function Tn(e, t, n, r, i) {
	let a = new $(), o = new $(), s = new $(), c = new $(), l, u = (e) => {
		l !== void 0 && (clearInterval(l), l = void 0, c.emit("state", {
			state: "stopped",
			reason: e
		}));
	}, d = new $(), f, p = (e) => {
		f && (f.unsubscribe?.(), f = void 0, d.emit("state", {
			state: "stopped",
			reason: e
		}));
	}, m = (e) => {
		f && d.emit("result", {
			playerId: "mock-player",
			sessionId: f.sessionId,
			seq: f.seq++,
			serverTimeMs: Date.now(),
			final: !0,
			alternatives: e.slice(0, 5)
		});
	}, h = /* @__PURE__ */ new Map(), g = /* @__PURE__ */ new Map();
	return {
		client: {
			mode: e,
			player: { ...t.player },
			launch: t.server ? {
				code: t.launchCode,
				serverUrl: t.server.url
			} : { code: t.launchCode },
			room: { ...t.room },
			ready: () => n("sdk.ready", {}),
			loading: { progress: (e) => n("loading.progress", { value: Math.min(1, Math.max(0, e)) }) },
			game: {
				started: (e = {}) => n("game.started", e),
				ended: (e = {}) => n("game.ended", e)
			},
			ui: {
				share: () => n("ui.share", {}),
				fullscreen: () => n("ui.fullscreen", {}),
				orientation: (e) => n("ui.orientation", { orientation: e }),
				pttButton: ({ visible: e }) => n("ui.pttButton", { visible: e }),
				on: (e, t) => o.on(e, t)
			},
			voice: {
				pushToTalk: (e) => n("voice.pushToTalk", { action: e }),
				request: () => n("voice.request", { kind: "speak" }),
				on: (e, t) => s.on(e, t)
			},
			pitch: {
				start: ({ mode: t = "ptt" } = {}) => {
					if (t !== "ptt" && t !== "hands_free") return;
					if (e === "platform") return n("pitch.start", { mode: t });
					if (l !== void 0) return;
					if (f) return c.emit("state", {
						state: "stopped",
						reason: "busy"
					});
					let r = Date.now(), i = 0;
					c.emit("state", { state: "started" }), l = setInterval(() => c.emit("pitch", Sn(i++, r, Date.now())), 50);
				},
				stop: () => e === "platform" ? n("pitch.stop", {}) : u("game"),
				on: (e, t) => c.on(e, t)
			},
			speech: {
				start: ({ mode: t = "ptt" } = {}) => {
					if (t !== "ptt" && t !== "hands_free") return;
					if (e === "platform") return n("speech.start", { mode: t });
					if (f) return;
					if (l !== void 0) return d.emit("state", {
						state: "stopped",
						reason: "busy"
					});
					let r = {
						sessionId: `mock-${Date.now()}`,
						seq: 0
					};
					f = r, d.emit("state", { state: "started" }), r.unsubscribe = i?.(m, (e) => f === r && p(e));
				},
				stop: () => e === "platform" ? n("speech.stop", {}) : p("game"),
				on: (e, t) => d.on(e, t),
				mockResult: (t) => {
					if (e !== "mock") return;
					let n = String(t).trim().slice(0, 100);
					m(n ? [{
						text: n,
						confidence: 1
					}] : []);
				}
			},
			points: {
				purchase: (t) => new Promise((r) => {
					if (e === "mock" || !/^pur_[A-Za-z0-9_-]{22}$/.test(t)) return r({ status: "failed" });
					h.get(t)?.("cancelled");
					let i = setTimeout(() => a("failed"), xn), a = (e) => {
						clearTimeout(i), h.delete(t), r({ status: e });
					};
					h.set(t, a), n("purchase.request", { purchaseId: t });
				}),
				joinPool: (t) => new Promise((r) => {
					if (e === "mock" || !/^pool_[A-Za-z0-9_-]{22}$/.test(t)) return r({ status: "failed" });
					g.get(t)?.("failed");
					let i = setTimeout(() => a("failed"), xn), a = (e) => {
						clearTimeout(i), g.delete(t), r({ status: e });
					};
					g.set(t, a), n("pool.request", { poolId: t });
				})
			},
			on: (e, t) => a.on(e, t),
			destroy: () => {
				u("closed"), p("closed"), r();
			}
		},
		dispatch: (e) => {
			switch (e.type) {
				case "app.pause": return a.emit("pause", {});
				case "app.resume": return a.emit("resume", {});
				case "app.close": return a.emit("close", {});
				case "audio.mute": return a.emit("mute", e.payload);
				case "room.members": return a.emit("members", e.payload);
				case "ui.insets": return o.emit("insets", e.payload);
				case "voice.state": return s.emit("state", e.payload);
				case "voice.speaking": return s.emit("speaking", e.payload);
				case "purchase.result": return h.get(e.payload.purchaseId)?.(e.payload.status);
				case "pool.result": return g.get(e.payload.poolId)?.(e.payload.status);
				case "pitch.state": return c.emit("state", e.payload);
				case "voice.pitch": return c.emit("pitch", e.payload);
				case "speech.state": return d.emit("state", e.payload);
				case "voice.speech": return d.emit("result", e.payload);
				default: return;
			}
		}
	};
}
function En() {
	return Tn("mock", {
		protocolVersion: 1,
		launchCode: `mock-${wn()}`,
		player: {
			name: "測試玩家",
			locale: "zh-TW"
		},
		room: { id: "mock-room" }
	}, () => {}, () => {}).client;
}
function Dn(e) {
	try {
		let t = new URL(e).searchParams, n = new URL(t.get("pt_dev")), r = t.get("pt_player") ?? "";
		return n.protocol === "http:" && [
			"127.0.0.1",
			"localhost",
			"[::1]"
		].includes(n.hostname) && /^[\w-]{1,32}$/.test(r) ? {
			origin: n.origin,
			player: r
		} : null;
	} catch {
		return null;
	}
}
async function On(e, t) {
	let n = {};
	try {
		let r = await e.fetch(`${t.origin}/dev/launch-code?player=${t.player}`);
		n = await r.json(), r.ok || (n = { error: n.error ?? r.status });
	} catch {
		n = { error: "unreachable" };
	}
	let { launchCode: r, name: i, roomId: a } = n;
	if (typeof r != "string" || typeof i != "string" || typeof a != "string") throw Error(`pt dev 本機假平台：${String(n.error ?? "bad_response")}`);
	let o = {
		protocolVersion: 1,
		launchCode: r,
		player: {
			name: i,
			locale: "zh-TW"
		},
		room: { id: a }
	}, s = Mn(n.serverUrl);
	return s && (o.server = { url: s }), Tn("mock", o, () => {}, () => {}, jn(e, t)).client;
}
function kn(e) {
	let { text: t, phrase: n, slots: r } = e ?? {};
	if (typeof t != "string" || t === "") return [];
	let i = {
		text: t.slice(0, 100),
		confidence: 1
	};
	if (typeof n == "string" && n !== "" && n.length <= 60 && r && typeof r == "object") {
		let e = {};
		for (let [t, n] of Object.entries(r)) /^[a-z][a-z0-9_]{0,15}$/.test(t) && (Number.isInteger(n) || typeof n == "string" && n.length <= 20) && (e[t] = n);
		i.phrase = n, i.slots = e;
	}
	return [i];
}
var An = [
	"game",
	"user",
	"denied",
	"not_authorized",
	"mic_denied",
	"unsupported",
	"background",
	"timeout",
	"busy",
	"closed"
];
function jn(e, t) {
	return (n, r) => {
		let i = !0, a = !1, o = async () => {
			if (!a) {
				a = !0;
				try {
					let a = await e.fetch(`${t.origin}/dev/speech?player=${t.player}${i ? "&start=1" : ""}`);
					i = !1;
					let o = await a.json();
					if (s) return;
					for (let e of Array.isArray(o.results) ? o.results : []) {
						let t = e.alternatives;
						Array.isArray(t) && n(t.flatMap((e) => kn(e)));
					}
					typeof o.stop == "string" && An.includes(o.stop) && r(o.stop);
				} catch {} finally {
					a = !1;
				}
			}
		}, s = !1, c = setInterval(() => void o(), 400);
		return o(), () => {
			s = !0, clearInterval(c);
		};
	};
}
function Mn(e) {
	if (typeof e != "string") return null;
	try {
		let t = new URL(e);
		return t.protocol === "http:" && [
			"127.0.0.1",
			"localhost",
			"[::1]"
		].includes(t.hostname) && t.pathname === "/" && !t.search && !t.hash ? t.origin : null;
	} catch {
		return null;
	}
}
function Nn(e) {
	let t = e.transport ?? bn(), n = Dn(t.href);
	if (n) return On(t, n);
	if (!t.embedded) return Promise.resolve(En());
	let r = e.platformOrigins.filter((e) => e !== "*");
	if (r.length === 0) return Promise.reject(/* @__PURE__ */ Error("platformOrigins 未設定：請填入玩聚平台的完整來源"));
	let i = e.timeoutMs ?? 5e3, a = e.helloIntervalMs ?? 250;
	return new Promise((e, n) => {
		let o = 0, s = null, c = "", l = "", u = null, d = (e, t, n) => (o += 1, {
			pt: "playtogether",
			v: 1,
			type: e,
			id: `g${o}`,
			...n ? {
				instance: c,
				nonce: l
			} : {},
			payload: t
		}), f = () => {
			let e = d("sdk.hello", { sdkVersion: yn }, !1);
			for (let n of r) t.post(e, n);
		}, p = () => {
			clearInterval(h), clearTimeout(g);
		}, m = t.listen((n) => {
			if (n.source !== t.parent || (s ? n.origin !== s : !r.includes(n.origin))) return;
			let i = mn(n.data);
			if (!i.ok) return;
			let a = i.value;
			if (s) a.instance === c && a.nonce === l && u?.(a);
			else {
				if (a.type !== "sdk.init" || !a.instance || !a.nonce) return;
				s = n.origin, c = a.instance, l = a.nonce, p();
				let r = Tn("platform", a.payload, (e, n) => {
					t.post(d(e, n, !0), s);
				}, m);
				u = r.dispatch, e(r.client);
			}
		}), h = setInterval(f, a), g = setTimeout(() => {
			p(), m(), n(new Cn());
		}, i);
		f();
	});
}
var Pn = { init: Nn };
export { Cn as HandshakeTimeoutError, Pn as PlayTogether, yn as SDK_VERSION, bn as browserTransport, Dn as devPlatform, Nn as init, Sn as mockPitchFrame };

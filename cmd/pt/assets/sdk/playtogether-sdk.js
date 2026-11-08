// ../../node_modules/zod/v4/core/util.js
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
function ee(e) {
	return Object.keys(e).filter((t) => e[t]._zod.optin !== void 0 && e[t]._zod.optout === "optional");
}
var te = {
	safeint: [-(2 ** 53 - 1), 2 ** 53 - 1],
	int32: [-2147483648, 2147483647],
	uint32: [0, 4294967295],
	float32: [-34028234663852886e22, 34028234663852886e22],
	float64: [-Number.MAX_VALUE, Number.MAX_VALUE]
};
function h(e, t = 0) {
	if (e.aborted === !0) return !0;
	for (let n = t; n < e.issues.length; n++) if (e.issues[n]?.continue !== !0) return !0;
	return !1;
}
function ne(e, t = 0) {
	if (e.aborted === !0) return !0;
	for (let n = t; n < e.issues.length; n++) if (e.issues[n]?.continue === !1) return !0;
	return !1;
}
function re(e, t) {
	return t.map((t) => {
		var n;
		return (n = t).path ?? (n.path = []), t.path.unshift(e), t;
	});
}
function g(e) {
	return typeof e == "string" ? e : e?.message;
}
function ie(e) {
	let t = [];
	for (let n of e ?? []) {
		let e = n._zod.def;
		if (!/_(length|size)$|_equals$/.test(e.check)) break;
		let r = e.maximum ?? e.length ?? e.size;
		r >= 0 && t.push([r, n]);
	}
	return t;
}
function _(e, t, n) {
	var r;
	for (let i = t; i < e.length; i++) (r = e[i]).schema ?? (r.schema = n);
}
function v(e, t, n) {
	var r;
	let i = e.inst?._zod?.traits;
	i?.has("$ZodType") && (i.has("$ZodCheck") ? (r = e).schema ?? (r.schema = e.inst) : e.schema = e.inst);
	let a = e.schema === e.inst ? void 0 : e.schema?._zod.def?.error, o = e.message ? e.message : g(e.inst?._zod.def?.error?.(e)) ?? g(a?.(e)) ?? g(t?.error?.(e)) ?? g(n.customError?.(e)) ?? g(n.localeError?.(e)) ?? "Invalid input", s = {};
	for (let t of Object.keys(e)) t !== "inst" && t !== "schema" && t !== "continue" && t !== "input" && t !== "__proto__" && (s[t] = e[t]);
	return s.path ?? (s.path = []), s.message = o, t?.reportInput && (s.input = e.input), s;
}
var ae = /[\uD800-\uDBFF]/;
function oe(e) {
	let t = e.length;
	if (!ae.test(e)) return t;
	let n = t;
	for (let r = 0; r < t - 1; r++) (e.charCodeAt(r) & 64512) == 55296 && (e.charCodeAt(r + 1) & 64512) == 56320 && (n--, r++);
	return n;
}
function se(e) {
	return Array.isArray(e) ? "array" : typeof e == "string" ? "string" : "unknown";
}
function ce(...e) {
	let [t, n, r] = e;
	return typeof t == "string" ? {
		message: t,
		code: "custom",
		input: n,
		inst: r
	} : { ...t };
}
function le(e, t) {
	for (let n in t) {
		let r = Object.getOwnPropertyDescriptor(t, n);
		r.get ? Object.defineProperty(e, n, {
			...r,
			enumerable: !1
		}) : de(e, n, r.value);
	}
}
function y(e, t, n, r = !0) {
	return Object.defineProperty(e, t, {
		configurable: !0,
		writable: !0,
		enumerable: r,
		value: n
	}), n;
}
function ue(e, t, n) {
	return y(e, t, n, !1);
}
function de(e, t, n) {
	Object.defineProperty(e, t, {
		configurable: !0,
		get() {
			return this == null ? n : y(this, t, n.bind(this));
		},
		set(e) {
			y(this, t, e);
		}
	});
}
function fe(e, t) {
	let n = Object.getPrototypeOf(e);
	return t in n ? void 0 : n;
}
var b, x = !1, pe = {
	configurable: !0,
	get() {
		x = !0;
	}
};
function S(e, t, n) {
	let r = Object.getPrototypeOf(e._zod);
	t in r && b !== e._zod ? b = void 0 : (b = e._zod, Object.defineProperty(r, t, {
		configurable: !0,
		get() {
			Object.defineProperty(this, t, pe);
			let e = x;
			x = !1;
			try {
				let r = n(this);
				return x ? delete this[t] : Object.defineProperty(this, t, {
					configurable: !0,
					writable: !0,
					value: r
				}), x = x || e, r;
			} catch (n) {
				throw delete this[t], x = x || e, n;
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
function me(e, t, n, r) {
	let i = fe(e, t);
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
// ../../node_modules/zod/v4/core/core.js
var he, C = {
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
			C.value = new a(r);
			try {
				Object.defineProperty(n, "_zod", C);
			} finally {
				C.value = void 0;
			}
		} else if (n._zod.traits.has(e)) return;
		if (n._zod.traits.add(e), t(n, r), s) {
			let e = Object.getPrototypeOf(n), t = n._zod.constr.prototype, r = e;
			for (; r && r !== t;) r = Object.getPrototypeOf(r);
			let i = r ?? e;
			s.has(i) || (s.add(i), le(i, o));
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
(he = globalThis).__zod_globalConfig ?? (he.__zod_globalConfig = {});
var E = globalThis.__zod_globalConfig;
function D(e) {
	return e && Object.assign(E, e), E;
}
// ../../node_modules/zod/v4/core/errors.js
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
}, O = {
	value: void 0,
	enumerable: !1
}, xe = /* @__PURE__ */ new WeakSet([Object.prototype, Error.prototype]), k = /* @__PURE__ */ w("$ZodError", (e, t) => {
	e.name = "$ZodError", O.value = t, Object.defineProperty(e, "issues", O), O.value = void 0, Object.defineProperty(e, "message", be);
	let n = Object.getPrototypeOf(e);
	xe.has(n) || (xe.add(n), Object.defineProperty(n, "toString", {
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
}, void 0, { Parent: Error }), Se = /* @__PURE__*/ ((e) => {
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
			let n = new ((a?.Err) ?? e)(c.issues.map((e) => v(e, o, D())));
			throw s(n, a?.callee ?? t), n;
		}
		return c.value;
	};
	return t;
})(k), Ce = /* @__PURE__*/ ((e) => {
	let t = async (n, r, i, a) => {
		let o = i ? {
			...i,
			async: !0
		} : { async: !0 }, c = n._zod.run({
			value: r,
			issues: []
		}, o);
		if (c instanceof Promise && (c = await c), c.issues.length) {
			let n = new ((a?.Err) ?? e)(c.issues.map((e) => v(e, o, D())));
			throw s(n, a?.callee ?? t), n;
		}
		return c.value;
	};
	return t;
})(k), we = /* @__PURE__*/ ((e) => (t, n, r) => {
	let i = r ? {
		...r,
		async: !1
	} : { async: !1 }, a = t._zod.run({
		value: n,
		issues: []
	}, i);
	if (a instanceof Promise) throw new T();
	return a.issues.length ? Te(e, a.issues, i) : {
		success: !0,
		data: a.value
	};
})(k);
function Te(e, t, n) {
	let r;
	return {
		success: !1,
		get error() {
			return r || (r = new e(t.map((e) => v(e, n, D()))), t = void 0, n = void 0), r;
		},
		set error(e) {
			r = e, t = void 0, n = void 0;
		}
	};
}
var Ee = /* @__PURE__*/ ((e) => async (t, n, r) => {
	let i = r ? {
		...r,
		async: !0
	} : { async: !0 }, a = t._zod.run({
		value: n,
		issues: []
	}, i);
	return a instanceof Promise && (a = await a), a.issues.length ? Te(e, a.issues, i) : {
		success: !0,
		data: a.value
	};
})(k), De = /^[\s\S]{0,}$/, Oe = /^-?\d+(?:\.\d+)?$/, ke = /^(?:true|false)$/i, A = /*@__PURE__*/ w("$ZodCheck", (e, t) => {
	var n;
	e._zod ?? (e._zod = {}), e._zod.def = t, (n = e._zod).onattach ?? (n.onattach = []);
}), Ae = (e) => {
	let t = e.value;
	return !i(t) && t.length !== void 0;
}, j = {
	number: "number",
	bigint: "bigint",
	object: "date"
}, je = /*@__PURE__*/ w("$ZodCheckLessThan", (e, t) => {
	A.init(e, t);
	let n = j[typeof t.value];
	e._zod.check = (r) => {
		(t.inclusive ? r.value <= t.value : r.value < t.value) || r.issues.push({
			origin: j[typeof r.value] ?? n,
			code: "too_big",
			maximum: typeof t.value == "object" ? t.value.getTime() : t.value,
			input: r.value,
			inclusive: t.inclusive,
			inst: e,
			continue: !t.abort
		});
	};
}), Me = /*@__PURE__*/ w("$ZodCheckGreaterThan", (e, t) => {
	A.init(e, t);
	let n = j[typeof t.value];
	e._zod.check = (r) => {
		(t.inclusive ? r.value >= t.value : r.value > t.value) || r.issues.push({
			origin: j[typeof r.value] ?? n,
			code: "too_small",
			minimum: typeof t.value == "object" ? t.value.getTime() : t.value,
			input: r.value,
			inclusive: t.inclusive,
			inst: e,
			continue: !t.abort
		});
	};
}), Ne = /*@__PURE__*/ w("$ZodCheckNumberFormat", (e, t) => {
	A.init(e, t), t.format = t.format || "float64";
	let n = t.format?.includes("int"), r = n ? "int" : "number", [i, a] = te[t.format];
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
}), Pe = /*@__PURE__*/ w("$ZodCheckMaxLength", (e, t) => {
	var n;
	A.init(e, t), (n = e._zod.def).when ?? (n.when = Ae), e._zod.check = (n) => {
		let r = n.value, i = r.length;
		if ((typeof r == "string" && i > t.maximum ? oe(r) : i) <= t.maximum) return;
		let a = se(r);
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
}), Fe = /*@__PURE__*/ w("$ZodCheckMinLength", (e, t) => {
	var n;
	A.init(e, t), (n = e._zod.def).when ?? (n.when = Ae), e._zod.check = (n) => {
		let r = n.value, i = r.length;
		if ((typeof r == "string" && i >= t.minimum && i < t.minimum * 2 ? oe(r) : i) >= t.minimum) return;
		let a = se(r);
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
}), Ie = /*@__PURE__*/ w("$ZodCheckStringFormat", (e, t) => {
	var n, r;
	A.init(e, t), t.pattern ? (n = e._zod).check ?? (n.check = (n) => {
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
}), Le = /*@__PURE__*/ w("$ZodCheckRegex", (e, t) => {
	Ie.init(e, t), e._zod.check = (n) => {
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
}), Re = {
	major: 4,
	minor: 6,
	patch: 7
}, M = /*@__PURE__*/ w("$ZodType", (e, t) => {
	var n;
	e ?? (e = {}), e._zod.def = t, e._zod.bag = e._zod.bag || {}, e._zod.version = Re;
	let r = e._zod.def.checks, i = e._zod.traits.has("$ZodCheck") ? [e, ...r ?? []] : r?.length ? [...r] : [];
	for (let t of i) for (let n of t._zod.onattach) n(e);
	if (i.length === 0) (n = e._zod).deferred ?? (n.deferred = []), e._zod.deferred?.push(() => {
		e._zod.run = e._zod.parse;
	});
	else {
		let t = (t, n, r) => {
			if (t.memo) return t;
			let i = h(t), a;
			for (let o of n) {
				if (o._zod.def.when) {
					if (ne(t) || !o._zod.def.when(t)) continue;
				} else if (i) continue;
				let n = t.issues.length, s = o._zod.check(t);
				if (s instanceof Promise && r?.async === !1) throw new T();
				if (a || s instanceof Promise) a = (a ?? Promise.resolve()).then(async () => {
					await s, t.issues.length !== n && (_(t.issues, n, e), i || (i = h(t, n)));
				});
				else {
					if (t.issues.length === n) continue;
					_(t.issues, n, e), i || (i = h(t, n));
				}
			}
			return a ? a.then(() => t) : t;
		}, n = (n, r, a) => {
			if (h(n)) return n.aborted = !0, n;
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
		return ue(this, "~standard", Ve(this));
	},
	set "~standard"(e) {
		y(this, "~standard", e);
	}
}), ze = (e, t) => e.issues.length ? { issues: e.issues.map((e) => v(e, t, D())) } : { value: e.value };
async function Be(e, t) {
	let n = { async: !0 };
	return ze(await e._zod.run({
		value: t,
		issues: []
	}, n), n);
}
function Ve(e) {
	return {
		validate: (t) => {
			let n = { async: !1 };
			try {
				let r = e._zod.run({
					value: t,
					issues: []
				}, n);
				if (!(r instanceof Promise)) return ze(r, n);
			} catch {}
			return Be(e, t);
		},
		vendor: "zod",
		version: 1
	};
}
var He = /*@__PURE__*/ w("$ZodString", (e, t) => {
	M.init(e, t), e._zod.pattern = t.pattern ?? De, e._zod.parse = (n, r) => {
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
}), Ue = /*@__PURE__*/ w("$ZodNumber", (e, t) => {
	M.init(e, t), e._zod.pattern = Oe, e._zod.parse = (n, r) => {
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
}), We = /*@__PURE__*/ w("$ZodNumberFormat", (e, t) => {
	Ne.init(e, t), Ue.init(e, t);
}), Ge = /*@__PURE__*/ w("$ZodBoolean", (e, t) => {
	M.init(e, t), e._zod.pattern = ke, e._zod.parse = (n, r) => {
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
}), Ke = /*@__PURE__*/ w("$ZodUnknown", (e, t) => {
	M.init(e, t), e._zod.parse = (e) => e;
});
function qe(e, t) {
	let n = ie(e._zod.def.checks);
	if (!n.length) return;
	let r = Math.min(...n.map((e) => e[0])), i = e._zod.parse;
	e._zod.parse = (n, a) => {
		let o = t(n.value);
		if (o > r && !a?.skipChecks && a?.direction !== "backward") {
			for (let [t, r] of ie(e._zod.def.checks)) if (o > t && r._zod.def.when(n)) {
				let t = n.issues.length;
				return r._zod.check(n), n.issues[t].continue = !1, _(n.issues, t, e), n;
			}
		}
		return i(n, a);
	};
}
var Je = (e) => Array.isArray(e) ? e.length : -1;
function Ye(e, t, n) {
	e.issues.length && t.issues.push(...re(n, e.issues)), t.value[n] = e.value;
}
var Xe = /*@__PURE__*/ w("$ZodArray", (e, t) => {
	M.init(e, t);
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
			if (c instanceof Promise) o.push(c.then((t) => Ye(t, r, e)));
			else if (Ye(c, r, e), s && c.issues.length !== 0 && h(c)) break;
		}
		return o.length ? Promise.all(o).then(() => r) : r;
	}, qe(e, Je);
});
function N(e, t, n, r, i, a) {
	let o = n in r, s = a === "optional";
	if (o || !s || i !== "optional") {
		if (e.issues.length) {
			if (i !== void 0 && s && !o) return;
			t.issues.push(...re(n, e.issues));
		}
		!o && i === void 0 ? e.issues.length || t.issues.push({
			code: "invalid_type",
			expected: "nonoptional",
			input: void 0,
			path: [n]
		}) : e.value === void 0 ? (o || i === "defaulted" && !s) && (t.value[n] = void 0) : t.value[n] = e.value;
	}
}
var Ze = [];
function Qe(e) {
	let t = Object.keys(e.shape), n = Object.getOwnPropertySymbols(e.shape), r = n.length ? n : Ze, i = r.length ? [...t, ...r] : t;
	for (let t of i) if (!e.shape?.[t]?._zod?.traits?.has("$ZodType")) throw Error(`Invalid element at key "${String(t)}": expected a Zod schema`);
	let a = ee(e.shape);
	return {
		...e,
		allKeys: i,
		symbolKeys: r,
		keySet: new Set(t),
		numKeys: t.length,
		optionalKeys: new Set(a)
	};
}
function $e(e, t, n, r, i, a, o) {
	let s = [], c = i.keySet, l = i.catchall._zod, u = l.def.type, d = l.optin, f = l.optout, p = 0;
	for (let i in t) {
		if (o && n.issues.length !== p) {
			if (h(n, p)) break;
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
		a instanceof Promise ? e.push(a.then((e) => N(e, n, i, t, d, f))) : N(a, n, i, t, d, f);
	}
	return s.length && n.issues.push({
		code: "unrecognized_keys",
		keys: s,
		input: t,
		inst: a,
		continue: !0
	}), e.length ? Promise.all(e).then(() => n) : n;
}
var et = /*@__PURE__*/ w("$ZodObject", (e, t) => {
	M.init(e, t);
	let n = Object.getOwnPropertyDescriptor(t, "shape"), i = n?.get ? n.get.raw : t.shape ?? {};
	if (i) {
		let e = () => {
			let n = { ...i };
			return Object.defineProperty(t, "shape", { value: n }), e.raw = n, n;
		};
		e.raw = i, Object.defineProperty(t, "shape", { get: e });
	}
	let a = r(() => Qe(t));
	S(e, "propValues", (e) => {
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
				if (h(t, f)) break;
				f = t.issues.length;
			}
			if (e === "__proto__") continue;
			let a = o[e], s = a._zod.optin, l = a._zod.optout, u = a._zod.run({
				value: r[e],
				issues: []
			}, n);
			u instanceof Promise ? i.push(u.then((n) => N(n, t, e, r, s, l))) : N(u, t, e, r, s, l);
		}
		return l ? $e(i, r, t, n, a.value, e, c === !0) : i.length ? Promise.all(i).then(() => t) : t;
	};
}), tt = /*@__PURE__*/ w("$ZodEnum", (t, n) => {
	M.init(t, n);
	let r = e(n.entries), i = new Set(r);
	t._zod.values = i, S(t, "pattern", (t) => {
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
}), nt = /*@__PURE__*/ w("$ZodLiteral", (e, t) => {
	M.init(e, t);
	let n = new Set(t.values);
	e._zod.values = n, S(e, "pattern", (e) => {
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
function rt(e, t) {
	return e.value = t.issues.length ? void 0 : t.value, e;
}
var it = /*@__PURE__*/ w("$ZodOptional", (e, t) => {
	M.init(e, t), S(e, "optin", (e) => e.def.innerType._zod.optin === "defaulted" ? "defaulted" : "optional"), e._zod.optout = "optional", S(e, "values", (e) => {
		let t = e.def.innerType._zod.values;
		return t ? /* @__PURE__ */ new Set([...t, void 0]) : void 0;
	}), S(e, "pattern", (e) => {
		let t = e.def.innerType._zod.pattern;
		return t ? RegExp(`^(${a(t.source)})?$`) : void 0;
	}), e._zod.parse = (e, n) => {
		if (e.value === void 0) {
			if (t.innerType._zod.optin !== "defaulted") return e;
			let r = t.innerType._zod.run({
				value: e.value,
				issues: []
			}, n);
			return r instanceof Promise ? r.then((t) => rt(e, t)) : rt(e, r);
		}
		return t.innerType._zod.run(e, n);
	};
}), at = /*@__PURE__*/ w("$ZodNullable", (e, t) => {
	M.init(e, t), S(e, "optin", (e) => e.def.innerType._zod.optin), S(e, "optout", (e) => e.def.innerType._zod.optout), S(e, "pattern", (e) => {
		let t = e.def.innerType._zod.pattern;
		return t ? RegExp(`^(${a(t.source)}|null)$`) : void 0;
	}), S(e, "values", (e) => e.def.innerType._zod.values ? /* @__PURE__ */ new Set([...e.def.innerType._zod.values, null]) : void 0), e._zod.parse = (e, n) => e.value === null ? e : t.innerType._zod.run(e, n);
}), ot = /*@__PURE__*/ w("$ZodDefault", (e, t) => {
	M.init(e, t), e._zod.optin = "defaulted", S(e, "values", (e) => e.def.innerType._zod.values), e._zod.parse = (e, n) => {
		if (n.direction === "backward") return t.innerType._zod.run(e, n);
		if (e.value === void 0) return e.value = t.defaultValue, e;
		let r = t.innerType._zod.run(e, n);
		return r instanceof Promise ? r.then((e) => st(e, t)) : st(r, t);
	};
});
function st(e, t) {
	return e.value === void 0 && (e.value = t.defaultValue), e;
}
var ct = /*@__PURE__*/ w("$ZodCustom", (e, t) => {
	A.init(e, t), M.init(e, t), e._zod.parse = (e, t) => e, e._zod.check = (n) => {
		let r = n.value, i = t.fn(r);
		if (i instanceof Promise) return i.then((t) => lt(t, n, r, e));
		lt(i, n, r, e);
	};
});
function lt(e, t, n, r) {
	if (!e) {
		let e = {
			code: "custom",
			input: n,
			inst: r,
			path: [...r._zod.def.path ?? []],
			continue: !r._zod.def.abort
		};
		r._zod.def.params && (e.params = r._zod.def.params), t.issues.push(ce(e));
	}
}
// ../../node_modules/zod/v4/core/api.js
function ut(e) {
	return e.checks && (e.checks = [...e.checks]), e;
}
// @__NO_SIDE_EFFECTS__
function dt(e, t) {
	return new e(ut({
		type: "string",
		...m(t)
	}));
}
// @__NO_SIDE_EFFECTS__
function ft(e, t) {
	return new e(ut({
		type: "number",
		checks: [],
		...m(t)
	}));
}
// @__NO_SIDE_EFFECTS__
function pt(e, t) {
	return new e({
		type: "number",
		check: "number_format",
		abort: !1,
		format: "safeint",
		...m(t)
	});
}
// @__NO_SIDE_EFFECTS__
function mt(e, t) {
	return new e({
		type: "boolean",
		...m(t)
	});
}
// @__NO_SIDE_EFFECTS__
function ht(e) {
	return new e({ type: "unknown" });
}
// @__NO_SIDE_EFFECTS__
function gt(e, t) {
	return new je({
		check: "less_than",
		...m(t),
		value: e,
		inclusive: !0
	});
}
// @__NO_SIDE_EFFECTS__
function _t(e, t) {
	return new Me({
		check: "greater_than",
		...m(t),
		value: e,
		inclusive: !1
	});
}
// @__NO_SIDE_EFFECTS__
function P(e, t) {
	return new Me({
		check: "greater_than",
		...m(t),
		value: e,
		inclusive: !0
	});
}
// @__NO_SIDE_EFFECTS__
function F(e) {
	return /* @__PURE__ */ _t(0, e);
}
// @__NO_SIDE_EFFECTS__
function I(e, t) {
	return new Pe({
		check: "max_length",
		...m(t),
		maximum: e
	});
}
// @__NO_SIDE_EFFECTS__
function L(e, t) {
	return new Fe({
		check: "min_length",
		...m(t),
		minimum: e
	});
}
// @__NO_SIDE_EFFECTS__
function vt(e, t) {
	return new Le({
		check: "string_format",
		format: "regex",
		...m(t),
		pattern: e
	});
}
// @__NO_SIDE_EFFECTS__
function yt(e, t, n) {
	return new e({
		type: "custom",
		check: "custom",
		fn: t,
		...m(n)
	});
}
// ../../node_modules/zod/v4/mini/schemas.js
var R = /*@__PURE__*/ w("ZodMiniType", (e, t) => {
	if (!e._zod) throw Error("Uninitialized schema in ZodMiniType.");
	M.init(e, t), e.def = t, e.type = t.type;
}, {
	get with() {
		return this.check;
	},
	set with(e) {
		y(this, "with", e);
	},
	parse(e, t) {
		return Se(this, e, t, { callee: this.parse });
	},
	parseAsync(e, t) {
		return Ce(this, e, t, { callee: this.parseAsync });
	},
	safeParse(e, t) {
		return we(this, e, t);
	},
	safeParseAsync(e, t) {
		return Ee(this, e, t);
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
}), bt = /*@__PURE__*/ w("ZodMiniString", (e, t) => {
	He.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function z(e) {
	return /* @__PURE__ */ dt(bt, e);
}
var xt = /*@__PURE__*/ w("ZodMiniNumber", (e, t) => {
	Ue.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function St(e) {
	return /* @__PURE__ */ ft(xt, e);
}
var Ct = /*@__PURE__*/ w("ZodMiniNumberFormat", (e, t) => {
	We.init(e, t), xt.init(e, t);
});
function B(e) {
	return /* @__PURE__ */ pt(Ct, e);
}
var wt = /*@__PURE__*/ w("ZodMiniBoolean", (e, t) => {
	Ge.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function V(e) {
	return /* @__PURE__ */ mt(wt, e);
}
var Tt = /*@__PURE__*/ w("ZodMiniUnknown", (e, t) => {
	Ke.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Et() {
	return /* @__PURE__ */ ht(Tt);
}
var Dt = /*@__PURE__*/ w("ZodMiniArray", (e, t) => {
	Xe.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function H(e, t) {
	return new Dt({
		type: "array",
		element: e,
		...m(t)
	});
}
var Ot = /*@__PURE__*/ w("ZodMiniObject", (e, t) => {
	et.init(e, t), R.init(e, t), me(e, "shape", (e) => e._zod.def.shape, !1);
});
// @__NO_SIDE_EFFECTS__
function U(e, t) {
	return new Ot({
		type: "object",
		shape: e ?? {},
		...m(t)
	});
}
var kt = /*@__PURE__*/ w("ZodMiniEnum", (e, t) => {
	tt.init(e, t), R.init(e, t), e.options = [...e._zod.values];
});
// @__NO_SIDE_EFFECTS__
function W(e, t) {
	return new kt({
		type: "enum",
		entries: Array.isArray(e) ? Object.fromEntries(e.map((e) => [e, e])) : e,
		...m(t)
	});
}
var At = /*@__PURE__*/ w("ZodMiniLiteral", (e, t) => {
	nt.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function G(e, t) {
	return new At({
		type: "literal",
		values: Array.isArray(e) ? e : [e],
		...m(t)
	});
}
var jt = /*@__PURE__*/ w("ZodMiniOptional", (e, t) => {
	it.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function K(e) {
	return new jt({
		type: "optional",
		innerType: e
	});
}
var Mt = /*@__PURE__*/ w("ZodMiniNullable", (e, t) => {
	at.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function q(e) {
	return new Mt({
		type: "nullable",
		innerType: e
	});
}
var Nt = /*@__PURE__*/ w("ZodMiniDefault", (e, t) => {
	ot.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Pt(e, t) {
	return new Nt({
		type: "default",
		innerType: e,
		get defaultValue() {
			return typeof t == "function" ? t() : u(t);
		}
	});
}
var Ft = /*@__PURE__*/ w("ZodMiniCustom", (e, t) => {
	ct.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function J(e, t = {}) {
	return /* @__PURE__ */ yt(Ft, e, t);
}
var Y = (/* @__PURE__ */ z()).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(128)), It = (/* @__PURE__ */ z()).check(/* @__PURE__ */ I(64)), Lt = (/* @__PURE__ */ z()).check(/* @__PURE__ */ I(256), /* @__PURE__ */ vt(/^https?:\/\/d-[a-z0-9]{16}\.[a-z0-9.-]+(:[0-9]{1,5})?$/)), Rt = (/* @__PURE__ */ z()).check(/* @__PURE__ */ vt(/^pur_[A-Za-z0-9_-]{22}$/)), X = (e, t) => (/* @__PURE__ */ St()).check(/* @__PURE__ */ P(e), /* @__PURE__ */ gt(t)), zt = /* @__PURE__ */ U({
	id: Y,
	name: It
}), Bt = /* @__PURE__ */ W([
	"open",
	"host_only",
	"turn",
	"allowlist"
]), Vt = /* @__PURE__ */ W(["ptt"]), Ht = /* @__PURE__ */ W(["self"]), Ut = /* @__PURE__ */ W([
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
]);
(/* @__PURE__ */ z()).check(/* @__PURE__ */ I(32)), X(0, 1), B().check(/* @__PURE__ */ P(0), /* @__PURE__ */ gt(1e3)), X(0, 86400);
var Wt = /* @__PURE__ */ U({
	top: X(0, 1e4),
	right: X(0, 1e4),
	bottom: X(0, 1e4),
	left: X(0, 1e4)
}), Gt = {
	"sdk.init": /* @__PURE__ */ U({
		protocolVersion: /* @__PURE__ */ G(1),
		launchCode: (/* @__PURE__ */ z()).check(/* @__PURE__ */ L(16), /* @__PURE__ */ I(256)),
		player: /* @__PURE__ */ U({
			name: It,
			locale: (/* @__PURE__ */ z()).check(/* @__PURE__ */ I(16))
		}),
		room: /* @__PURE__ */ U({ id: Y }),
		server: /* @__PURE__ */ K(/* @__PURE__ */ U({ url: Lt }))
	}),
	"app.pause": /* @__PURE__ */ U({}),
	"app.resume": /* @__PURE__ */ U({}),
	"app.close": /* @__PURE__ */ U({}),
	"audio.mute": /* @__PURE__ */ U({ muted: /* @__PURE__ */ V() }),
	"room.members": /* @__PURE__ */ U({ members: (/* @__PURE__ */ H(zt)).check(/* @__PURE__ */ I(64)) }),
	"ui.insets": Wt,
	"voice.state": /* @__PURE__ */ U({
		connected: /* @__PURE__ */ V(),
		canSpeak: /* @__PURE__ */ V(),
		mode: Bt
	}),
	"voice.speaking": /* @__PURE__ */ U({
		playerId: Y,
		level: X(0, 1)
	}),
	"purchase.result": /* @__PURE__ */ U({
		purchaseId: Rt,
		status: /* @__PURE__ */ W([
			"completed",
			"cancelled",
			"failed"
		])
	}),
	"pitch.state": /* @__PURE__ */ U({
		state: /* @__PURE__ */ W(["started", "stopped"]),
		reason: /* @__PURE__ */ K(Ut)
	}),
	"voice.pitch": /* @__PURE__ */ U({
		playerId: Y,
		sessionId: Y,
		seq: B().check(/* @__PURE__ */ P(0)),
		t: X(0, 0xe8d4a51000),
		serverTimeMs: X(0, 0x5af3107a4000),
		voiced: /* @__PURE__ */ V(),
		hz: /* @__PURE__ */ q(X(20, 5e3)),
		note: /* @__PURE__ */ q(X(0, 127)),
		cents: /* @__PURE__ */ q(X(-50, 50)),
		confidence: X(0, 1)
	})
}, Kt = /* @__PURE__ */ U({
	pt: /* @__PURE__ */ G("playtogether"),
	v: /* @__PURE__ */ G(1),
	type: (/* @__PURE__ */ z()).check(/* @__PURE__ */ I(64)),
	id: (/* @__PURE__ */ z()).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(64)),
	instance: /* @__PURE__ */ K((/* @__PURE__ */ z()).check(/* @__PURE__ */ I(64))),
	nonce: /* @__PURE__ */ K((/* @__PURE__ */ z()).check(/* @__PURE__ */ I(128))),
	payload: /* @__PURE__ */ Et()
});
function qt(e) {
	try {
		return new TextEncoder().encode(JSON.stringify(e)).length;
	} catch {
		return Infinity;
	}
}
function Jt(e) {
	if (qt(e) > 8192) return {
		ok: !1,
		error: "too_large"
	};
	let t = Kt.safeParse(e);
	return t.success ? {
		ok: !0,
		value: t.data
	} : {
		ok: !1,
		error: "bad_envelope"
	};
}
function Yt(e) {
	let t = Jt(e);
	if (!t.ok) return t;
	let n = Gt[t.value.type];
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
(/* @__PURE__ */ U({
	roomId: Y,
	gameInstanceId: Y,
	sequence: B().check(/* @__PURE__ */ P(0)),
	mode: Bt,
	allowedSpeakerIds: /* @__PURE__ */ Pt((/* @__PURE__ */ H(Y)).check(/* @__PURE__ */ I(64)), []),
	effectiveAtServerMs: B().check(/* @__PURE__ */ F()),
	expiresAtServerMs: B().check(/* @__PURE__ */ F())
})).check(/* @__PURE__ */ J((e) => e.expiresAtServerMs > e.effectiveAtServerMs, "expiresAtServerMs 必須晚於 effectiveAtServerMs"), /* @__PURE__ */ J((e) => e.mode !== "turn" || e.allowedSpeakerIds.length === 1, "turn 模式必須剛好一位發言者"), /* @__PURE__ */ J((e) => e.mode !== "allowlist" || e.allowedSpeakerIds.length >= 1, "allowlist 模式至少要一位發言者"), /* @__PURE__ */ J((e) => e.mode !== "open" && e.mode !== "host_only" || e.allowedSpeakerIds.length === 0, "open / host_only 模式不帶發言者名單"), /* @__PURE__ */ J((e) => new Set(e.allowedSpeakerIds).size === e.allowedSpeakerIds.length, "發言者名單不得重複")), (/* @__PURE__ */ U({
	roomId: Y,
	gameInstanceId: Y,
	epoch: B().check(/* @__PURE__ */ P(0)),
	sequence: B().check(/* @__PURE__ */ P(0)),
	allowedPlayerIds: (/* @__PURE__ */ H(Y)).check(/* @__PURE__ */ L(1), /* @__PURE__ */ I(64)),
	mode: Vt,
	audience: Ht,
	effectiveAtServerMs: B().check(/* @__PURE__ */ F()),
	expiresAtServerMs: B().check(/* @__PURE__ */ F())
})).check(/* @__PURE__ */ J((e) => e.expiresAtServerMs > e.effectiveAtServerMs, "expiresAtServerMs 必須晚於 effectiveAtServerMs"), /* @__PURE__ */ J((e) => new Set(e.allowedPlayerIds).size === e.allowedPlayerIds.length, "名單不得重複")), (/* @__PURE__ */ z()).check(/* @__PURE__ */ L(16), /* @__PURE__ */ I(256)), B().check(/* @__PURE__ */ P(0)), (/* @__PURE__ */ H((/* @__PURE__ */ z()).check(/* @__PURE__ */ I(64)))).check(/* @__PURE__ */ I(500));
// \0@oxc-project+runtime@0.156.0/helpers/esm/typeof.js
function Z(e) {
	"@babel/helpers - typeof";
	return Z = typeof Symbol == "function" && typeof Symbol.iterator == "symbol" ? function(e) {
		return typeof e;
	} : function(e) {
		return e && typeof Symbol == "function" && e.constructor === Symbol && e !== Symbol.prototype ? "symbol" : typeof e;
	}, Z(e);
}
// \0@oxc-project+runtime@0.156.0/helpers/esm/toPrimitive.js
function Xt(e, t) {
	if (Z(e) != "object" || !e) return e;
	var n = e[Symbol.toPrimitive];
	if (n !== void 0) {
		var r = n.call(e, t || "default");
		if (Z(r) != "object") return r;
		throw TypeError("@@toPrimitive must return a primitive value.");
	}
	return (t === "string" ? String : Number)(e);
}
// \0@oxc-project+runtime@0.156.0/helpers/esm/toPropertyKey.js
function Zt(e) {
	var t = Xt(e, "string");
	return Z(t) == "symbol" ? t : t + "";
}
// \0@oxc-project+runtime@0.156.0/helpers/esm/defineProperty.js
function Qt(e, t, n) {
	return (t = Zt(t)) in e ? Object.defineProperty(e, t, {
		value: n,
		enumerable: !0,
		configurable: !0,
		writable: !0
	}) : e[t] = n, e;
}
// src/index.ts
var $t = "0.1.0";
function en(e = window) {
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
var tn = 15e4, Q = class {
	constructor() {
		Qt(this, "map", /* @__PURE__ */ new Map());
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
function nn(e, t, n) {
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
var rn = class extends Error {
	constructor() {
		super("無法和玩聚平台完成握手（逾時）"), this.name = "HandshakeTimeoutError";
	}
};
function an() {
	let e = /* @__PURE__ */ new Uint8Array(16);
	return crypto.getRandomValues(e), Array.from(e, (e) => e.toString(16).padStart(2, "0")).join("");
}
function $(e, t, n, r) {
	let i = new Q(), a = new Q(), o = new Q(), s = new Q(), c, l = (e) => {
		c !== void 0 && (clearInterval(c), c = void 0, s.emit("state", {
			state: "stopped",
			reason: e
		}));
	}, u = /* @__PURE__ */ new Map();
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
				on: (e, t) => a.on(e, t)
			},
			voice: {
				pushToTalk: (e) => n("voice.pushToTalk", { action: e }),
				request: () => n("voice.request", { kind: "speak" }),
				on: (e, t) => o.on(e, t)
			},
			pitch: {
				start: ({ mode: t = "ptt" } = {}) => {
					if (t !== "ptt") return;
					if (e === "platform") return n("pitch.start", { mode: t });
					if (c !== void 0) return;
					let r = Date.now(), i = 0;
					s.emit("state", { state: "started" }), c = setInterval(() => s.emit("pitch", nn(i++, r, Date.now())), 50);
				},
				stop: () => e === "platform" ? n("pitch.stop", {}) : l("game"),
				on: (e, t) => s.on(e, t)
			},
			points: { purchase: (t) => new Promise((r) => {
				if (e === "mock" || !/^pur_[A-Za-z0-9_-]{22}$/.test(t)) return r({ status: "failed" });
				u.get(t)?.("cancelled");
				let i = setTimeout(() => a("failed"), tn), a = (e) => {
					clearTimeout(i), u.delete(t), r({ status: e });
				};
				u.set(t, a), n("purchase.request", { purchaseId: t });
			}) },
			on: (e, t) => i.on(e, t),
			destroy: () => {
				l("closed"), r();
			}
		},
		dispatch: (e) => {
			switch (e.type) {
				case "app.pause": return i.emit("pause", {});
				case "app.resume": return i.emit("resume", {});
				case "app.close": return i.emit("close", {});
				case "audio.mute": return i.emit("mute", e.payload);
				case "room.members": return i.emit("members", e.payload);
				case "ui.insets": return a.emit("insets", e.payload);
				case "voice.state": return o.emit("state", e.payload);
				case "voice.speaking": return o.emit("speaking", e.payload);
				case "purchase.result": return u.get(e.payload.purchaseId)?.(e.payload.status);
				case "pitch.state": return s.emit("state", e.payload);
				case "voice.pitch": return s.emit("pitch", e.payload);
				default: return;
			}
		}
	};
}
function on() {
	return $("mock", {
		protocolVersion: 1,
		launchCode: `mock-${an()}`,
		player: {
			name: "測試玩家",
			locale: "zh-TW"
		},
		room: { id: "mock-room" }
	}, () => {}, () => {}).client;
}
function sn(e) {
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
async function cn(e, t) {
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
	}, s = ln(n.serverUrl);
	return s && (o.server = { url: s }), $("mock", o, () => {}, () => {}).client;
}
function ln(e) {
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
function un(e) {
	let t = e.transport ?? en();
	if (!t.embedded) {
		let e = sn(t.href);
		return e ? cn(t, e) : Promise.resolve(on());
	}
	let n = e.platformOrigins.filter((e) => e !== "*");
	if (n.length === 0) return Promise.reject(/* @__PURE__ */ Error("platformOrigins 未設定：請填入玩聚平台的完整來源"));
	let r = e.timeoutMs ?? 5e3, i = e.helloIntervalMs ?? 250;
	return new Promise((e, a) => {
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
			let e = d("sdk.hello", { sdkVersion: $t }, !1);
			for (let r of n) t.post(e, r);
		}, p = () => {
			clearInterval(ee), clearTimeout(te);
		}, m = t.listen((r) => {
			if (r.source !== t.parent || (s ? r.origin !== s : !n.includes(r.origin))) return;
			let i = Yt(r.data);
			if (!i.ok) return;
			let a = i.value;
			if (s) a.instance === c && a.nonce === l && u?.(a);
			else {
				if (a.type !== "sdk.init" || !a.instance || !a.nonce) return;
				s = r.origin, c = a.instance, l = a.nonce, p();
				let n = $("platform", a.payload, (e, n) => {
					t.post(d(e, n, !0), s);
				}, m);
				u = n.dispatch, e(n.client);
			}
		}), ee = setInterval(f, i), te = setTimeout(() => {
			p(), m(), a(new rn());
		}, r);
		f();
	});
}
var dn = { init: un };
export { rn as HandshakeTimeoutError, dn as PlayTogether, $t as SDK_VERSION, en as browserTransport, sn as devPlatform, un as init, nn as mockPitchFrame };

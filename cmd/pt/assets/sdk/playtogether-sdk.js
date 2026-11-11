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
function h(e) {
	return Object.keys(e).filter((t) => e[t]._zod.optin !== void 0 && e[t]._zod.optout === "optional");
}
var ee = {
	safeint: [-(2 ** 53 - 1), 2 ** 53 - 1],
	int32: [-2147483648, 2147483647],
	uint32: [0, 4294967295],
	float32: [-34028234663852886e22, 34028234663852886e22],
	float64: [-Number.MAX_VALUE, Number.MAX_VALUE]
};
function g(e, t = 0) {
	if (e.aborted === !0) return !0;
	for (let n = t; n < e.issues.length; n++) if (e.issues[n]?.continue !== !0) return !0;
	return !1;
}
function te(e, t = 0) {
	if (e.aborted === !0) return !0;
	for (let n = t; n < e.issues.length; n++) if (e.issues[n]?.continue === !1) return !0;
	return !1;
}
function _(e, t) {
	return t.map((t) => {
		var n;
		return (n = t).path ?? (n.path = []), t.path.unshift(e), t;
	});
}
function v(e) {
	return typeof e == "string" ? e : e?.message;
}
function ne(e) {
	let t = [];
	for (let n of e ?? []) {
		let e = n._zod.def;
		if (!/_(length|size)$|_equals$/.test(e.check)) break;
		let r = e.maximum ?? e.length ?? e.size;
		r >= 0 && t.push([r, n]);
	}
	return t;
}
function re(e, t, n) {
	var r;
	for (let i = t; i < e.length; i++) (r = e[i]).schema ?? (r.schema = n);
}
function y(e, t, n) {
	var r;
	let i = e.inst?._zod?.traits;
	i?.has("$ZodType") && (i.has("$ZodCheck") ? (r = e).schema ?? (r.schema = e.inst) : e.schema = e.inst);
	let a = e.schema === e.inst ? void 0 : e.schema?._zod.def?.error, o = e.message ? e.message : v(e.inst?._zod.def?.error?.(e)) ?? v(a?.(e)) ?? v(t?.error?.(e)) ?? v(n.customError?.(e)) ?? v(n.localeError?.(e)) ?? "Invalid input", s = {};
	for (let t of Object.keys(e)) t !== "inst" && t !== "schema" && t !== "continue" && t !== "input" && t !== "__proto__" && (s[t] = e[t]);
	return s.path ?? (s.path = []), s.message = o, t?.reportInput && (s.input = e.input), s;
}
var ie = /[\uD800-\uDBFF]/;
function ae(e) {
	let t = e.length;
	if (!ie.test(e)) return t;
	let n = t;
	for (let r = 0; r < t - 1; r++) (e.charCodeAt(r) & 64512) == 55296 && (e.charCodeAt(r + 1) & 64512) == 56320 && (n--, r++);
	return n;
}
function oe(e) {
	return Array.isArray(e) ? "array" : typeof e == "string" ? "string" : "unknown";
}
function se(...e) {
	let [t, n, r] = e;
	return typeof t == "string" ? {
		message: t,
		code: "custom",
		input: n,
		inst: r
	} : { ...t };
}
function ce(e, t) {
	for (let n in t) {
		let r = Object.getOwnPropertyDescriptor(t, n);
		r.get ? Object.defineProperty(e, n, {
			...r,
			enumerable: !1
		}) : ue(e, n, r.value);
	}
}
function b(e, t, n, r = !0) {
	return Object.defineProperty(e, t, {
		configurable: !0,
		writable: !0,
		enumerable: r,
		value: n
	}), n;
}
function le(e, t, n) {
	return b(e, t, n, !1);
}
function ue(e, t, n) {
	Object.defineProperty(e, t, {
		configurable: !0,
		get() {
			return this == null ? n : b(this, t, n.bind(this));
		},
		set(e) {
			b(this, t, e);
		}
	});
}
function de(e, t) {
	let n = Object.getPrototypeOf(e);
	return t in n ? void 0 : n;
}
var fe, x = !1, pe = {
	configurable: !0,
	get() {
		x = !0;
	}
};
function S(e, t, n) {
	let r = Object.getPrototypeOf(e._zod);
	t in r && fe !== e._zod ? fe = void 0 : (fe = e._zod, Object.defineProperty(r, t, {
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
	let i = de(e, t);
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
var he, ge = {
	value: void 0,
	enumerable: !1
}, _e = "captureStackTrace" in Error ? Error : null;
function ve(e) {
	let t = _e;
	if (t) {
		let n = t.stackTraceLimit;
		if (typeof n == "number") {
			try {
				t.stackTraceLimit = 0;
			} catch {
				return _e = null, new e();
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
function C(e, t, n, r) {
	let i = {};
	function a(e) {
		this.def = e, this.constr = d, this.traits = /* @__PURE__ */ new Set();
	}
	a.prototype = i;
	let o = n, s = o && /* @__PURE__ */ new WeakSet();
	function c(n, r) {
		if (!n._zod) {
			ge.value = new a(r);
			try {
				Object.defineProperty(n, "_zod", ge);
			} finally {
				ge.value = void 0;
			}
		} else if (n._zod.traits.has(e)) return;
		if (n._zod.traits.add(e), t(n, r), s) {
			let e = Object.getPrototypeOf(n), t = n._zod.constr.prototype, r = e;
			for (; r && r !== t;) r = Object.getPrototypeOf(r);
			let i = r ?? e;
			s.has(i) || (s.add(i), ce(i, o));
		}
		let i = d.prototype;
		for (let e in i) Object.prototype.hasOwnProperty.call(i, e) && (e in n || (n[e] = i[e].bind(n)));
	}
	let l = r?.Parent ?? Object;
	class u extends l {}
	Object.defineProperty(u, "name", { value: e });
	function d(e) {
		let t = r?.Parent ? ve(u) : this;
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
var w = class extends Error {
	constructor() {
		super("Encountered Promise during synchronous parse. Use .parseAsync() instead.");
	}
};
(he = globalThis).__zod_globalConfig ?? (he.__zod_globalConfig = {});
var T = globalThis.__zod_globalConfig;
function E(e) {
	return e && Object.assign(T, e), T;
}
// ../../node_modules/zod/v4/core/errors.js
function ye() {
	let e = this._zod;
	return e.message ?? (e.message = JSON.stringify(e.def, t, 2)), e.message;
}
function be(e) {
	this._zod.message = e;
}
var xe = {
	get: ye,
	set: be,
	enumerable: !0,
	configurable: !0
}, Se = {
	value: void 0,
	enumerable: !1
}, Ce = /* @__PURE__ */ new WeakSet([Object.prototype, Error.prototype]), D = /* @__PURE__ */ C("$ZodError", (e, t) => {
	e.name = "$ZodError", Se.value = t, Object.defineProperty(e, "issues", Se), Se.value = void 0, Object.defineProperty(e, "message", xe);
	let n = Object.getPrototypeOf(e);
	Ce.has(n) || (Ce.add(n), Object.defineProperty(n, "toString", {
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
}, void 0, { Parent: Error }), we = /* @__PURE__*/ ((e) => {
	let t = (n, r, i, a) => {
		let o = i ? {
			...i,
			async: !1
		} : { async: !1 }, c = n._zod.run({
			value: r,
			issues: []
		}, o);
		if (c instanceof Promise) throw new w();
		if (c.issues.length) {
			let n = new ((a?.Err) ?? e)(c.issues.map((e) => y(e, o, E())));
			throw s(n, a?.callee ?? t), n;
		}
		return c.value;
	};
	return t;
})(D), Te = /* @__PURE__*/ ((e) => {
	let t = async (n, r, i, a) => {
		let o = i ? {
			...i,
			async: !0
		} : { async: !0 }, c = n._zod.run({
			value: r,
			issues: []
		}, o);
		if (c instanceof Promise && (c = await c), c.issues.length) {
			let n = new ((a?.Err) ?? e)(c.issues.map((e) => y(e, o, E())));
			throw s(n, a?.callee ?? t), n;
		}
		return c.value;
	};
	return t;
})(D), Ee = /* @__PURE__*/ ((e) => (t, n, r) => {
	let i = r ? {
		...r,
		async: !1
	} : { async: !1 }, a = t._zod.run({
		value: n,
		issues: []
	}, i);
	if (a instanceof Promise) throw new w();
	return a.issues.length ? De(e, a.issues, i) : {
		success: !0,
		data: a.value
	};
})(D);
function De(e, t, n) {
	let r;
	return {
		success: !1,
		get error() {
			return r || (r = new e(t.map((e) => y(e, n, E()))), t = void 0, n = void 0), r;
		},
		set error(e) {
			r = e, t = void 0, n = void 0;
		}
	};
}
var Oe = /* @__PURE__*/ ((e) => async (t, n, r) => {
	let i = r ? {
		...r,
		async: !0
	} : { async: !0 }, a = t._zod.run({
		value: n,
		issues: []
	}, i);
	return a instanceof Promise && (a = await a), a.issues.length ? De(e, a.issues, i) : {
		success: !0,
		data: a.value
	};
})(D), ke = /^[\s\S]{0,}$/, Ae = /^-?\d+(?:\.\d+)?$/, je = /^(?:true|false)$/i, O = /*@__PURE__*/ C("$ZodCheck", (e, t) => {
	var n;
	e._zod ?? (e._zod = {}), e._zod.def = t, (n = e._zod).onattach ?? (n.onattach = []);
}), Me = (e) => {
	let t = e.value;
	return !i(t) && t.length !== void 0;
}, k = {
	number: "number",
	bigint: "bigint",
	object: "date"
}, Ne = /*@__PURE__*/ C("$ZodCheckLessThan", (e, t) => {
	O.init(e, t);
	let n = k[typeof t.value];
	e._zod.check = (r) => {
		(t.inclusive ? r.value <= t.value : r.value < t.value) || r.issues.push({
			origin: k[typeof r.value] ?? n,
			code: "too_big",
			maximum: typeof t.value == "object" ? t.value.getTime() : t.value,
			input: r.value,
			inclusive: t.inclusive,
			inst: e,
			continue: !t.abort
		});
	};
}), Pe = /*@__PURE__*/ C("$ZodCheckGreaterThan", (e, t) => {
	O.init(e, t);
	let n = k[typeof t.value];
	e._zod.check = (r) => {
		(t.inclusive ? r.value >= t.value : r.value > t.value) || r.issues.push({
			origin: k[typeof r.value] ?? n,
			code: "too_small",
			minimum: typeof t.value == "object" ? t.value.getTime() : t.value,
			input: r.value,
			inclusive: t.inclusive,
			inst: e,
			continue: !t.abort
		});
	};
}), Fe = /*@__PURE__*/ C("$ZodCheckNumberFormat", (e, t) => {
	O.init(e, t), t.format = t.format || "float64";
	let n = t.format?.includes("int"), r = n ? "int" : "number", [i, a] = ee[t.format];
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
}), Ie = /*@__PURE__*/ C("$ZodCheckMaxLength", (e, t) => {
	var n;
	O.init(e, t), (n = e._zod.def).when ?? (n.when = Me), e._zod.check = (n) => {
		let r = n.value, i = r.length;
		if ((typeof r == "string" && i > t.maximum ? ae(r) : i) <= t.maximum) return;
		let a = oe(r);
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
}), Le = /*@__PURE__*/ C("$ZodCheckMinLength", (e, t) => {
	var n;
	O.init(e, t), (n = e._zod.def).when ?? (n.when = Me), e._zod.check = (n) => {
		let r = n.value, i = r.length;
		if ((typeof r == "string" && i >= t.minimum && i < t.minimum * 2 ? ae(r) : i) >= t.minimum) return;
		let a = oe(r);
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
}), Re = /*@__PURE__*/ C("$ZodCheckStringFormat", (e, t) => {
	var n, r;
	O.init(e, t), t.pattern ? (n = e._zod).check ?? (n.check = (n) => {
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
}), ze = /*@__PURE__*/ C("$ZodCheckRegex", (e, t) => {
	Re.init(e, t), e._zod.check = (n) => {
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
}), Be = {
	major: 4,
	minor: 6,
	patch: 7
}, A = /*@__PURE__*/ C("$ZodType", (e, t) => {
	var n;
	e ?? (e = {}), e._zod.def = t, e._zod.bag = e._zod.bag || {}, e._zod.version = Be;
	let r = e._zod.def.checks, i = e._zod.traits.has("$ZodCheck") ? [e, ...r ?? []] : r?.length ? [...r] : [];
	for (let t of i) for (let n of t._zod.onattach) n(e);
	if (i.length === 0) (n = e._zod).deferred ?? (n.deferred = []), e._zod.deferred?.push(() => {
		e._zod.run = e._zod.parse;
	});
	else {
		let t = (t, n, r) => {
			if (t.memo) return t;
			let i = g(t), a;
			for (let o of n) {
				if (o._zod.def.when) {
					if (te(t) || !o._zod.def.when(t)) continue;
				} else if (i) continue;
				let n = t.issues.length, s = o._zod.check(t);
				if (s instanceof Promise && r?.async === !1) throw new w();
				if (a || s instanceof Promise) a = (a ?? Promise.resolve()).then(async () => {
					await s, t.issues.length !== n && (re(t.issues, n, e), i || (i = g(t, n)));
				});
				else {
					if (t.issues.length === n) continue;
					re(t.issues, n, e), i || (i = g(t, n));
				}
			}
			return a ? a.then(() => t) : t;
		}, n = (n, r, a) => {
			if (g(n)) return n.aborted = !0, n;
			let o = t(r, i, a);
			if (o instanceof Promise) {
				if (a.async === !1) throw new w();
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
				if (a.async === !1) throw new w();
				return o.then((e) => t(e, i, a));
			}
			return t(o, i, a);
		};
	}
}, {
	get "~standard"() {
		return le(this, "~standard", Ue(this));
	},
	set "~standard"(e) {
		b(this, "~standard", e);
	}
}), Ve = (e, t) => e.issues.length ? { issues: e.issues.map((e) => y(e, t, E())) } : { value: e.value };
async function He(e, t) {
	let n = { async: !0 };
	return Ve(await e._zod.run({
		value: t,
		issues: []
	}, n), n);
}
function Ue(e) {
	return {
		validate: (t) => {
			let n = { async: !1 };
			try {
				let r = e._zod.run({
					value: t,
					issues: []
				}, n);
				if (!(r instanceof Promise)) return Ve(r, n);
			} catch {}
			return He(e, t);
		},
		vendor: "zod",
		version: 1
	};
}
var We = /*@__PURE__*/ C("$ZodString", (e, t) => {
	A.init(e, t), e._zod.pattern = t.pattern ?? ke, e._zod.parse = (n, r) => {
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
}), Ge = /*@__PURE__*/ C("$ZodNumber", (e, t) => {
	A.init(e, t), e._zod.pattern = Ae, e._zod.parse = (n, r) => {
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
}), Ke = /*@__PURE__*/ C("$ZodNumberFormat", (e, t) => {
	Fe.init(e, t), Ge.init(e, t);
}), qe = /*@__PURE__*/ C("$ZodBoolean", (e, t) => {
	A.init(e, t), e._zod.pattern = je, e._zod.parse = (n, r) => {
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
}), Je = /*@__PURE__*/ C("$ZodUnknown", (e, t) => {
	A.init(e, t), e._zod.parse = (e) => e;
});
function Ye(e, t) {
	let n = ne(e._zod.def.checks);
	if (!n.length) return;
	let r = Math.min(...n.map((e) => e[0])), i = e._zod.parse;
	e._zod.parse = (n, a) => {
		let o = t(n.value);
		if (o > r && !a?.skipChecks && a?.direction !== "backward") {
			for (let [t, r] of ne(e._zod.def.checks)) if (o > t && r._zod.def.when(n)) {
				let t = n.issues.length;
				return r._zod.check(n), n.issues[t].continue = !1, re(n.issues, t, e), n;
			}
		}
		return i(n, a);
	};
}
var Xe = (e) => Array.isArray(e) ? e.length : -1;
function Ze(e, t, n) {
	e.issues.length && t.issues.push(..._(n, e.issues)), t.value[n] = e.value;
}
var Qe = /*@__PURE__*/ C("$ZodArray", (e, t) => {
	A.init(e, t);
	let n = T.memoizer;
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
			if (c instanceof Promise) o.push(c.then((t) => Ze(t, r, e)));
			else if (Ze(c, r, e), s && c.issues.length !== 0 && g(c)) break;
		}
		return o.length ? Promise.all(o).then(() => r) : r;
	}, Ye(e, Xe);
});
function j(e, t, n, r, i, a) {
	let o = n in r, s = a === "optional";
	if (o || !s || i !== "optional") {
		if (e.issues.length) {
			if (i !== void 0 && s && !o) return;
			t.issues.push(..._(n, e.issues));
		}
		!o && i === void 0 ? e.issues.length || t.issues.push({
			code: "invalid_type",
			expected: "nonoptional",
			input: void 0,
			path: [n]
		}) : e.value === void 0 ? (o || i === "defaulted" && !s) && (t.value[n] = void 0) : t.value[n] = e.value;
	}
}
var $e = [];
function et(e) {
	let t = Object.keys(e.shape), n = Object.getOwnPropertySymbols(e.shape), r = n.length ? n : $e, i = r.length ? [...t, ...r] : t;
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
function tt(e, t, n, r, i, a, o) {
	let s = [], c = i.keySet, l = i.catchall._zod, u = l.def.type, d = l.optin, f = l.optout, p = 0;
	for (let i in t) {
		if (o && n.issues.length !== p) {
			if (g(n, p)) break;
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
		a instanceof Promise ? e.push(a.then((e) => j(e, n, i, t, d, f))) : j(a, n, i, t, d, f);
	}
	return s.length && n.issues.push({
		code: "unrecognized_keys",
		keys: s,
		input: t,
		inst: a,
		continue: !0
	}), e.length ? Promise.all(e).then(() => n) : n;
}
var nt = /*@__PURE__*/ C("$ZodObject", (e, t) => {
	A.init(e, t);
	let n = Object.getOwnPropertyDescriptor(t, "shape"), i = n?.get ? n.get.raw : t.shape ?? {};
	if (i) {
		let e = () => {
			let n = { ...i };
			return Object.defineProperty(t, "shape", { value: n }), e.raw = n, n;
		};
		e.raw = i, Object.defineProperty(t, "shape", { get: e });
	}
	let a = r(() => et(t));
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
	let s = c, l = t.catchall, u, d = T.memoizer;
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
				if (g(t, f)) break;
				f = t.issues.length;
			}
			if (e === "__proto__") continue;
			let a = o[e], s = a._zod.optin, l = a._zod.optout, u = a._zod.run({
				value: r[e],
				issues: []
			}, n);
			u instanceof Promise ? i.push(u.then((n) => j(n, t, e, r, s, l))) : j(u, t, e, r, s, l);
		}
		return l ? tt(i, r, t, n, a.value, e, c === !0) : i.length ? Promise.all(i).then(() => t) : t;
	};
});
function rt(e, t, n, r) {
	for (let n of e) if (n.issues.length === 0) return t.value = n.value, t;
	let i = e.filter((e) => !g(e));
	return i.length === 1 ? (t.value = i[0].value, i[0]) : (t.issues.push({
		code: "invalid_union",
		input: t.value,
		inst: n,
		errors: e.map((e) => e.issues.map((e) => y(e, r, E())))
	}), t);
}
var it = /*@__PURE__*/ C("$ZodUnion", (e, t) => {
	A.init(e, t), S(e, "optin", (e) => e.def.options.some((e) => e._zod.optin === "defaulted") ? "defaulted" : e.def.options.some((e) => e._zod.optin !== void 0) ? "optional" : void 0), S(e, "optout", (e) => e.def.options.some((e) => e._zod.optout === "optional") ? "optional" : void 0), S(e, "values", (e) => {
		if (e.def.options.every((e) => e._zod.values)) return new Set(e.def.options.flatMap((e) => Array.from(e._zod.values)));
	}), S(e, "pattern", (e) => {
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
		return a ? Promise.all(o).then((t) => rt(t, r, e, i)) : rt(o, r, e, i);
	};
}), at = /*@__PURE__*/ C("$ZodRecord", (e, t) => {
	A.init(e, t);
	let n = T.memoizer;
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
						issues: s.issues.map((e) => y(e, i, E())),
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
					e.issues.length && r.issues.push(..._(n, e.issues)), r.value[l] = e.value;
				})) : (u.issues.length && r.issues.push(..._(n, u.issues)), r.value[l] = u.value);
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
				if (typeof n == "string" && Ae.test(n) && l.issues.length) {
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
						issues: l.issues.map((e) => y(e, i, E())),
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
					e.issues.length && r.issues.push(..._(n, e.issues)), r.value[u] = e.value;
				})) : (d.issues.length && r.issues.push(..._(n, d.issues)), r.value[u] = d.value);
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
}), ot = /*@__PURE__*/ C("$ZodEnum", (t, n) => {
	A.init(t, n);
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
}), st = /*@__PURE__*/ C("$ZodLiteral", (e, t) => {
	A.init(e, t);
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
function ct(e, t) {
	return e.value = t.issues.length ? void 0 : t.value, e;
}
var lt = /*@__PURE__*/ C("$ZodOptional", (e, t) => {
	A.init(e, t), S(e, "optin", (e) => e.def.innerType._zod.optin === "defaulted" ? "defaulted" : "optional"), e._zod.optout = "optional", S(e, "values", (e) => {
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
			return r instanceof Promise ? r.then((t) => ct(e, t)) : ct(e, r);
		}
		return t.innerType._zod.run(e, n);
	};
}), ut = /*@__PURE__*/ C("$ZodNullable", (e, t) => {
	A.init(e, t), S(e, "optin", (e) => e.def.innerType._zod.optin), S(e, "optout", (e) => e.def.innerType._zod.optout), S(e, "pattern", (e) => {
		let t = e.def.innerType._zod.pattern;
		return t ? RegExp(`^(${a(t.source)}|null)$`) : void 0;
	}), S(e, "values", (e) => e.def.innerType._zod.values ? /* @__PURE__ */ new Set([...e.def.innerType._zod.values, null]) : void 0), e._zod.parse = (e, n) => e.value === null ? e : t.innerType._zod.run(e, n);
}), dt = /*@__PURE__*/ C("$ZodDefault", (e, t) => {
	A.init(e, t), e._zod.optin = "defaulted", S(e, "values", (e) => e.def.innerType._zod.values), e._zod.parse = (e, n) => {
		if (n.direction === "backward") return t.innerType._zod.run(e, n);
		if (e.value === void 0) return e.value = t.defaultValue, e;
		let r = t.innerType._zod.run(e, n);
		return r instanceof Promise ? r.then((e) => ft(e, t)) : ft(r, t);
	};
});
function ft(e, t) {
	return e.value === void 0 && (e.value = t.defaultValue), e;
}
var pt = /*@__PURE__*/ C("$ZodCustom", (e, t) => {
	O.init(e, t), A.init(e, t), e._zod.parse = (e, t) => e, e._zod.check = (n) => {
		let r = n.value, i = t.fn(r);
		if (i instanceof Promise) return i.then((t) => mt(t, n, r, e));
		mt(i, n, r, e);
	};
});
function mt(e, t, n, r) {
	if (!e) {
		let e = {
			code: "custom",
			input: n,
			inst: r,
			path: [...r._zod.def.path ?? []],
			continue: !r._zod.def.abort
		};
		r._zod.def.params && (e.params = r._zod.def.params), t.issues.push(se(e));
	}
}
// ../../node_modules/zod/v4/core/api.js
function ht(e) {
	return e.checks && (e.checks = [...e.checks]), e;
}
// @__NO_SIDE_EFFECTS__
function gt(e, t) {
	return new e(ht({
		type: "string",
		...m(t)
	}));
}
// @__NO_SIDE_EFFECTS__
function _t(e, t) {
	return new e(ht({
		type: "number",
		checks: [],
		...m(t)
	}));
}
// @__NO_SIDE_EFFECTS__
function vt(e, t) {
	return new e({
		type: "number",
		check: "number_format",
		abort: !1,
		format: "safeint",
		...m(t)
	});
}
// @__NO_SIDE_EFFECTS__
function yt(e, t) {
	return new e({
		type: "boolean",
		...m(t)
	});
}
// @__NO_SIDE_EFFECTS__
function bt(e) {
	return new e({ type: "unknown" });
}
// @__NO_SIDE_EFFECTS__
function M(e, t) {
	return new Ne({
		check: "less_than",
		...m(t),
		value: e,
		inclusive: !0
	});
}
// @__NO_SIDE_EFFECTS__
function xt(e, t) {
	return new Pe({
		check: "greater_than",
		...m(t),
		value: e,
		inclusive: !1
	});
}
// @__NO_SIDE_EFFECTS__
function N(e, t) {
	return new Pe({
		check: "greater_than",
		...m(t),
		value: e,
		inclusive: !0
	});
}
// @__NO_SIDE_EFFECTS__
function P(e) {
	return /* @__PURE__ */ xt(0, e);
}
// @__NO_SIDE_EFFECTS__
function F(e, t) {
	return new Ie({
		check: "max_length",
		...m(t),
		maximum: e
	});
}
// @__NO_SIDE_EFFECTS__
function I(e, t) {
	return new Le({
		check: "min_length",
		...m(t),
		minimum: e
	});
}
// @__NO_SIDE_EFFECTS__
function L(e, t) {
	return new ze({
		check: "string_format",
		format: "regex",
		...m(t),
		pattern: e
	});
}
// @__NO_SIDE_EFFECTS__
function St(e, t, n) {
	return new e({
		type: "custom",
		check: "custom",
		fn: t,
		...m(n)
	});
}
// ../../node_modules/zod/v4/mini/schemas.js
var R = /*@__PURE__*/ C("ZodMiniType", (e, t) => {
	if (!e._zod) throw Error("Uninitialized schema in ZodMiniType.");
	A.init(e, t), e.def = t, e.type = t.type;
}, {
	get with() {
		return this.check;
	},
	set with(e) {
		b(this, "with", e);
	},
	parse(e, t) {
		return we(this, e, t, { callee: this.parse });
	},
	parseAsync(e, t) {
		return Te(this, e, t, { callee: this.parseAsync });
	},
	safeParse(e, t) {
		return Ee(this, e, t);
	},
	safeParseAsync(e, t) {
		return Oe(this, e, t);
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
}), Ct = /*@__PURE__*/ C("ZodMiniString", (e, t) => {
	We.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function z(e) {
	return /* @__PURE__ */ gt(Ct, e);
}
var wt = /*@__PURE__*/ C("ZodMiniNumber", (e, t) => {
	Ge.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Tt(e) {
	return /* @__PURE__ */ _t(wt, e);
}
var Et = /*@__PURE__*/ C("ZodMiniNumberFormat", (e, t) => {
	Ke.init(e, t), wt.init(e, t);
});
function B(e) {
	return /* @__PURE__ */ vt(Et, e);
}
var Dt = /*@__PURE__*/ C("ZodMiniBoolean", (e, t) => {
	qe.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function V(e) {
	return /* @__PURE__ */ yt(Dt, e);
}
var Ot = /*@__PURE__*/ C("ZodMiniUnknown", (e, t) => {
	Je.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function kt() {
	return /* @__PURE__ */ bt(Ot);
}
var At = /*@__PURE__*/ C("ZodMiniArray", (e, t) => {
	Qe.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function H(e, t) {
	return new At({
		type: "array",
		element: e,
		...m(t)
	});
}
var jt = /*@__PURE__*/ C("ZodMiniObject", (e, t) => {
	nt.init(e, t), R.init(e, t), me(e, "shape", (e) => e._zod.def.shape, !1);
});
// @__NO_SIDE_EFFECTS__
function U(e, t) {
	return new jt({
		type: "object",
		shape: e ?? {},
		...m(t)
	});
}
var Mt = /*@__PURE__*/ C("ZodMiniUnion", (e, t) => {
	it.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Nt(e, t) {
	return new Mt({
		type: "union",
		options: e,
		...m(t)
	});
}
var Pt = /*@__PURE__*/ C("ZodMiniRecord", (e, t) => {
	at.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Ft(e, t, n) {
	return !t || !t._zod ? new Pt({
		type: "record",
		keyType: /* @__PURE__ */ z(),
		valueType: e,
		...m(t)
	}) : new Pt({
		type: "record",
		keyType: e,
		valueType: t,
		...m(n)
	});
}
var It = /*@__PURE__*/ C("ZodMiniEnum", (e, t) => {
	ot.init(e, t), R.init(e, t), e.options = [...e._zod.values];
});
// @__NO_SIDE_EFFECTS__
function W(e, t) {
	return new It({
		type: "enum",
		entries: Array.isArray(e) ? Object.fromEntries(e.map((e) => [e, e])) : e,
		...m(t)
	});
}
var Lt = /*@__PURE__*/ C("ZodMiniLiteral", (e, t) => {
	st.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function G(e, t) {
	return new Lt({
		type: "literal",
		values: Array.isArray(e) ? e : [e],
		...m(t)
	});
}
var Rt = /*@__PURE__*/ C("ZodMiniOptional", (e, t) => {
	lt.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function K(e) {
	return new Rt({
		type: "optional",
		innerType: e
	});
}
var zt = /*@__PURE__*/ C("ZodMiniNullable", (e, t) => {
	ut.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function q(e) {
	return new zt({
		type: "nullable",
		innerType: e
	});
}
var Bt = /*@__PURE__*/ C("ZodMiniDefault", (e, t) => {
	dt.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Vt(e, t) {
	return new Bt({
		type: "default",
		innerType: e,
		get defaultValue() {
			return typeof t == "function" ? t() : u(t);
		}
	});
}
var Ht = /*@__PURE__*/ C("ZodMiniCustom", (e, t) => {
	pt.init(e, t), R.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function J(e, t = {}) {
	return /* @__PURE__ */ St(Ht, e, t);
}
var Y = (/* @__PURE__ */ z()).check(/* @__PURE__ */ I(1), /* @__PURE__ */ F(128)), Ut = (/* @__PURE__ */ z()).check(/* @__PURE__ */ F(64)), Wt = (/* @__PURE__ */ z()).check(/* @__PURE__ */ F(256), /* @__PURE__ */ L(/^https?:\/\/d-[a-z0-9]{16}\.[a-z0-9.-]+(:[0-9]{1,5})?$/)), Gt = (/* @__PURE__ */ z()).check(/* @__PURE__ */ L(/^pur_[A-Za-z0-9_-]{22}$/)), X = (e, t) => (/* @__PURE__ */ Tt()).check(/* @__PURE__ */ N(e), /* @__PURE__ */ M(t)), Kt = /* @__PURE__ */ U({
	id: Y,
	name: Ut
}), qt = /* @__PURE__ */ W([
	"open",
	"host_only",
	"turn",
	"allowlist"
]), Jt = /* @__PURE__ */ W(["ptt"]), Yt = /* @__PURE__ */ W(["self"]), Xt = /* @__PURE__ */ W([
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
]), Zt = /* @__PURE__ */ W(["ptt"]), Qt = /* @__PURE__ */ W(["self"]), $t = /* @__PURE__ */ W(["zh-TW"]), en = Xt, tn = 5e3, nn = /^[a-z][a-z0-9_]{0,15}$/, rn = (/* @__PURE__ */ z()).check(/* @__PURE__ */ I(1), /* @__PURE__ */ F(60), /* @__PURE__ */ L(/^[^\u0000-\u001f\u007f]+$/)), an = /* @__PURE__ */ Nt([/* @__PURE__ */ U({
	type: /* @__PURE__ */ G("integer"),
	min: B().check(/* @__PURE__ */ N(0), /* @__PURE__ */ M(9999)),
	max: B().check(/* @__PURE__ */ N(0), /* @__PURE__ */ M(9999))
}), /* @__PURE__ */ U({
	type: /* @__PURE__ */ G("enum"),
	values: (/* @__PURE__ */ H((/* @__PURE__ */ z()).check(/* @__PURE__ */ I(1), /* @__PURE__ */ F(20), /* @__PURE__ */ L(/^[^\s{}\u0000-\u001f\u007f]+$/)))).check(/* @__PURE__ */ I(1), /* @__PURE__ */ F(200))
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
var cn = (/* @__PURE__ */ U({
	phrases: (/* @__PURE__ */ H(rn)).check(/* @__PURE__ */ I(1), /* @__PURE__ */ F(50)),
	slots: /* @__PURE__ */ K(/* @__PURE__ */ Ft((/* @__PURE__ */ z()).check(/* @__PURE__ */ L(nn)), an)),
	strict: /* @__PURE__ */ V()
})).check(/* @__PURE__ */ J((e) => Object.keys(e.slots ?? {}).length <= 8, "欄位最多 8 個"), /* @__PURE__ */ J((e) => new Set(e.phrases).size === e.phrases.length, "句型不得重複"), /* @__PURE__ */ J((e) => {
	let t = sn(e);
	return t >= 1 && t <= 5e3;
}, `句型用到的欄位要有定義，展開後最多 ${tn} 句`));
(/* @__PURE__ */ z()).check(/* @__PURE__ */ F(32)), X(0, 1), B().check(/* @__PURE__ */ N(0), /* @__PURE__ */ M(1e3)), X(0, 86400);
var ln = /* @__PURE__ */ U({
	top: X(0, 1e4),
	right: X(0, 1e4),
	bottom: X(0, 1e4),
	left: X(0, 1e4)
}), un = {
	"sdk.init": /* @__PURE__ */ U({
		protocolVersion: /* @__PURE__ */ G(1),
		launchCode: (/* @__PURE__ */ z()).check(/* @__PURE__ */ I(16), /* @__PURE__ */ F(256)),
		player: /* @__PURE__ */ U({
			name: Ut,
			locale: (/* @__PURE__ */ z()).check(/* @__PURE__ */ F(16))
		}),
		room: /* @__PURE__ */ U({ id: Y }),
		server: /* @__PURE__ */ K(/* @__PURE__ */ U({ url: Wt }))
	}),
	"app.pause": /* @__PURE__ */ U({}),
	"app.resume": /* @__PURE__ */ U({}),
	"app.close": /* @__PURE__ */ U({}),
	"audio.mute": /* @__PURE__ */ U({ muted: /* @__PURE__ */ V() }),
	"room.members": /* @__PURE__ */ U({ members: (/* @__PURE__ */ H(Kt)).check(/* @__PURE__ */ F(64)) }),
	"ui.insets": ln,
	"voice.state": /* @__PURE__ */ U({
		connected: /* @__PURE__ */ V(),
		canSpeak: /* @__PURE__ */ V(),
		mode: qt
	}),
	"voice.speaking": /* @__PURE__ */ U({
		playerId: Y,
		level: X(0, 1)
	}),
	"purchase.result": /* @__PURE__ */ U({
		purchaseId: Gt,
		status: /* @__PURE__ */ W([
			"completed",
			"cancelled",
			"failed"
		])
	}),
	"pitch.state": /* @__PURE__ */ U({
		state: /* @__PURE__ */ W(["started", "stopped"]),
		reason: /* @__PURE__ */ K(Xt)
	}),
	"voice.pitch": /* @__PURE__ */ U({
		playerId: Y,
		sessionId: Y,
		seq: B().check(/* @__PURE__ */ N(0)),
		t: X(0, 0xe8d4a51000),
		serverTimeMs: X(0, 0x5af3107a4000),
		voiced: /* @__PURE__ */ V(),
		hz: /* @__PURE__ */ q(X(20, 5e3)),
		note: /* @__PURE__ */ q(X(0, 127)),
		cents: /* @__PURE__ */ q(X(-50, 50)),
		confidence: X(0, 1)
	}),
	"speech.state": /* @__PURE__ */ U({
		state: /* @__PURE__ */ W(["started", "stopped"]),
		reason: /* @__PURE__ */ K(en)
	}),
	"voice.speech": /* @__PURE__ */ U({
		playerId: Y,
		sessionId: Y,
		seq: B().check(/* @__PURE__ */ N(0)),
		serverTimeMs: X(0, 0x5af3107a4000),
		final: /* @__PURE__ */ V(),
		alternatives: (/* @__PURE__ */ H(/* @__PURE__ */ U({
			text: (/* @__PURE__ */ z()).check(/* @__PURE__ */ I(1), /* @__PURE__ */ F(100)),
			confidence: /* @__PURE__ */ q(X(0, 1)),
			phrase: /* @__PURE__ */ K(rn),
			slots: /* @__PURE__ */ K(/* @__PURE__ */ Ft((/* @__PURE__ */ z()).check(/* @__PURE__ */ L(nn)), /* @__PURE__ */ Nt([B(), (/* @__PURE__ */ z()).check(/* @__PURE__ */ F(20))])))
		}))).check(/* @__PURE__ */ F(5))
	})
}, dn = /* @__PURE__ */ U({
	pt: /* @__PURE__ */ G("playtogether"),
	v: /* @__PURE__ */ G(1),
	type: (/* @__PURE__ */ z()).check(/* @__PURE__ */ F(64)),
	id: (/* @__PURE__ */ z()).check(/* @__PURE__ */ I(1), /* @__PURE__ */ F(64)),
	instance: /* @__PURE__ */ K((/* @__PURE__ */ z()).check(/* @__PURE__ */ F(64))),
	nonce: /* @__PURE__ */ K((/* @__PURE__ */ z()).check(/* @__PURE__ */ F(128))),
	payload: /* @__PURE__ */ kt()
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
(/* @__PURE__ */ U({
	roomId: Y,
	gameInstanceId: Y,
	sequence: B().check(/* @__PURE__ */ N(0)),
	mode: qt,
	allowedSpeakerIds: /* @__PURE__ */ Vt((/* @__PURE__ */ H(Y)).check(/* @__PURE__ */ F(64)), []),
	effectiveAtServerMs: B().check(/* @__PURE__ */ P()),
	expiresAtServerMs: B().check(/* @__PURE__ */ P())
})).check(/* @__PURE__ */ J((e) => e.expiresAtServerMs > e.effectiveAtServerMs, "expiresAtServerMs 必須晚於 effectiveAtServerMs"), /* @__PURE__ */ J((e) => e.mode !== "turn" || e.allowedSpeakerIds.length === 1, "turn 模式必須剛好一位發言者"), /* @__PURE__ */ J((e) => e.mode !== "allowlist" || e.allowedSpeakerIds.length >= 1, "allowlist 模式至少要一位發言者"), /* @__PURE__ */ J((e) => e.mode !== "open" && e.mode !== "host_only" || e.allowedSpeakerIds.length === 0, "open / host_only 模式不帶發言者名單"), /* @__PURE__ */ J((e) => new Set(e.allowedSpeakerIds).size === e.allowedSpeakerIds.length, "發言者名單不得重複")), (/* @__PURE__ */ U({
	roomId: Y,
	gameInstanceId: Y,
	epoch: B().check(/* @__PURE__ */ N(0)),
	sequence: B().check(/* @__PURE__ */ N(0)),
	allowedPlayerIds: (/* @__PURE__ */ H(Y)).check(/* @__PURE__ */ I(1), /* @__PURE__ */ F(64)),
	mode: Jt,
	audience: Yt,
	effectiveAtServerMs: B().check(/* @__PURE__ */ P()),
	expiresAtServerMs: B().check(/* @__PURE__ */ P())
})).check(/* @__PURE__ */ J((e) => e.expiresAtServerMs > e.effectiveAtServerMs, "expiresAtServerMs 必須晚於 effectiveAtServerMs"), /* @__PURE__ */ J((e) => new Set(e.allowedPlayerIds).size === e.allowedPlayerIds.length, "名單不得重複")), (/* @__PURE__ */ U({
	roomId: Y,
	gameInstanceId: Y,
	epoch: B().check(/* @__PURE__ */ N(0)),
	sequence: B().check(/* @__PURE__ */ N(0)),
	allowedPlayerIds: (/* @__PURE__ */ H(Y)).check(/* @__PURE__ */ I(1), /* @__PURE__ */ F(64)),
	mode: Zt,
	audience: Qt,
	language: $t,
	vocabulary: /* @__PURE__ */ K(cn),
	effectiveAtServerMs: B().check(/* @__PURE__ */ P()),
	expiresAtServerMs: B().check(/* @__PURE__ */ P())
})).check(/* @__PURE__ */ J((e) => e.expiresAtServerMs > e.effectiveAtServerMs, "expiresAtServerMs 必須晚於 effectiveAtServerMs"), /* @__PURE__ */ J((e) => new Set(e.allowedPlayerIds).size === e.allowedPlayerIds.length, "名單不得重複")), (/* @__PURE__ */ z()).check(/* @__PURE__ */ I(16), /* @__PURE__ */ F(256)), B().check(/* @__PURE__ */ N(0)), (/* @__PURE__ */ H((/* @__PURE__ */ z()).check(/* @__PURE__ */ F(64)))).check(/* @__PURE__ */ F(500));
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
function hn(e, t) {
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
function gn(e) {
	var t = hn(e, "string");
	return Z(t) == "symbol" ? t : t + "";
}
// \0@oxc-project+runtime@0.156.0/helpers/esm/defineProperty.js
function _n(e, t, n) {
	return (t = gn(t)) in e ? Object.defineProperty(e, t, {
		value: n,
		enumerable: !0,
		configurable: !0,
		writable: !0
	}) : e[t] = n, e;
}
// src/index.ts
var vn = "0.1.0";
function yn(e = window) {
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
var bn = 15e4, Q = class {
	constructor() {
		_n(this, "map", /* @__PURE__ */ new Map());
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
function xn(e, t, n) {
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
var Sn = class extends Error {
	constructor() {
		super("無法和玩聚平台完成握手（逾時）"), this.name = "HandshakeTimeoutError";
	}
};
function Cn() {
	let e = /* @__PURE__ */ new Uint8Array(16);
	return crypto.getRandomValues(e), Array.from(e, (e) => e.toString(16).padStart(2, "0")).join("");
}
function $(e, t, n, r, i) {
	let a = new Q(), o = new Q(), s = new Q(), c = new Q(), l, u = (e) => {
		l !== void 0 && (clearInterval(l), l = void 0, c.emit("state", {
			state: "stopped",
			reason: e
		}));
	}, d = new Q(), f, p = (e) => {
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
	}, h = /* @__PURE__ */ new Map();
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
					if (t !== "ptt") return;
					if (e === "platform") return n("pitch.start", { mode: t });
					if (l !== void 0) return;
					if (f) return c.emit("state", {
						state: "stopped",
						reason: "busy"
					});
					let r = Date.now(), i = 0;
					c.emit("state", { state: "started" }), l = setInterval(() => c.emit("pitch", xn(i++, r, Date.now())), 50);
				},
				stop: () => e === "platform" ? n("pitch.stop", {}) : u("game"),
				on: (e, t) => c.on(e, t)
			},
			speech: {
				start: ({ mode: t = "ptt" } = {}) => {
					if (t !== "ptt") return;
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
			points: { purchase: (t) => new Promise((r) => {
				if (e === "mock" || !/^pur_[A-Za-z0-9_-]{22}$/.test(t)) return r({ status: "failed" });
				h.get(t)?.("cancelled");
				let i = setTimeout(() => a("failed"), bn), a = (e) => {
					clearTimeout(i), h.delete(t), r({ status: e });
				};
				h.set(t, a), n("purchase.request", { purchaseId: t });
			}) },
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
				case "pitch.state": return c.emit("state", e.payload);
				case "voice.pitch": return c.emit("pitch", e.payload);
				case "speech.state": return d.emit("state", e.payload);
				case "voice.speech": return d.emit("result", e.payload);
				default: return;
			}
		}
	};
}
function wn() {
	return $("mock", {
		protocolVersion: 1,
		launchCode: `mock-${Cn()}`,
		player: {
			name: "測試玩家",
			locale: "zh-TW"
		},
		room: { id: "mock-room" }
	}, () => {}, () => {}).client;
}
function Tn(e) {
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
async function En(e, t) {
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
	}, s = An(n.serverUrl);
	return s && (o.server = { url: s }), $("mock", o, () => {}, () => {}, kn(e, t)).client;
}
function Dn(e) {
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
var On = [
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
function kn(e, t) {
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
						Array.isArray(t) && n(t.flatMap((e) => Dn(e)));
					}
					typeof o.stop == "string" && On.includes(o.stop) && r(o.stop);
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
function An(e) {
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
function jn(e) {
	let t = e.transport ?? yn(), n = Tn(t.href);
	if (n) return En(t, n);
	if (!t.embedded) return Promise.resolve(wn());
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
			let e = d("sdk.hello", { sdkVersion: vn }, !1);
			for (let n of r) t.post(e, n);
		}, p = () => {
			clearInterval(h), clearTimeout(ee);
		}, m = t.listen((n) => {
			if (n.source !== t.parent || (s ? n.origin !== s : !r.includes(n.origin))) return;
			let i = mn(n.data);
			if (!i.ok) return;
			let a = i.value;
			if (s) a.instance === c && a.nonce === l && u?.(a);
			else {
				if (a.type !== "sdk.init" || !a.instance || !a.nonce) return;
				s = n.origin, c = a.instance, l = a.nonce, p();
				let r = $("platform", a.payload, (e, n) => {
					t.post(d(e, n, !0), s);
				}, m);
				u = r.dispatch, e(r.client);
			}
		}), h = setInterval(f, a), ee = setTimeout(() => {
			p(), m(), n(new Sn());
		}, i);
		f();
	});
}
var Mn = { init: jn };
export { Sn as HandshakeTimeoutError, Mn as PlayTogether, vn as SDK_VERSION, yn as browserTransport, Tn as devPlatform, jn as init, xn as mockPitchFrame };

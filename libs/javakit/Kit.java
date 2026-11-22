// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.Consumer;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Kit — shared runtime for every RailShack Java microservice. JDK only, no
 * frameworks: health and readiness probes, Prometheus metrics, JSON logs, a
 * tiny JSON encoder/decoder and the X-Trace header the UI animates.
 */
public class Kit {
    public static final String BRAND = "RailShack by DevOps Shack";
    public static final String WEBSITE = "https://devopsshack.com";
    public static final String YOUTUBE = "https://www.youtube.com/@devopsshack";
    public static final String SOURCE = "https://github.com/jaiswaladi246/50-MicroServices-Project-By-DevOpsShack";
    public static final String POWERED_BY = "RailShack by DevOps Shack (devopsshack.com)";
    public final String name;
    private final List<Object[]> routes = new ArrayList<>();
    private final Map<String, AtomicLong> reqs = new ConcurrentHashMap<>();
    private final long started = System.currentTimeMillis();
    private final int work = intEnv("WORK_FACTOR", 0);
    private final int latency = intEnv("LATENCY_MS", 0);
    private final double failRate = Double.parseDouble(env("FAIL_RATE", "0"));
    private double durSum = 0;
    private long durCount = 0;

    public Kit(String name) {
        this.name = name;
        get("/api/" + name + "/info", c -> c.json(200, info()));
    }

    static String env(String k, String d) {
        String v = System.getenv(k);
        return v == null || v.isEmpty() ? d : v;
    }

    static int intEnv(String k, int d) {
        try { return Integer.parseInt(env(k, "" + d)); } catch (Exception e) { return d; }
    }

    static String pod() {
        String p = System.getenv("POD_NAME");
        if (p != null && !p.isEmpty()) return p;
        try { return InetAddress.getLocalHost().getHostName(); } catch (Exception e) { return "unknown"; }
    }

    public Map<String, Object> info() {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("service", name);
        m.put("version", env("SERVICE_VERSION", "1.0.0"));
        m.put("language", "java");
        m.put("pod", pod());
        m.put("uptimeSeconds", (System.currentTimeMillis() - started) / 1000);
        m.put("project", BRAND);
        m.put("website", WEBSITE);
        m.put("youtube", YOUTUBE);
        m.put("source", SOURCE);
        m.put("license", "Apache-2.0");
        return m;
    }

    public static class Ctx {
        public final Map<String, String> params = new HashMap<>();
        public final Map<String, String> query = new HashMap<>();
        public Object body;
        final HttpExchange ex;
        final Kit kit;
        int code = 200;

        Ctx(Kit kit, HttpExchange ex) { this.kit = kit; this.ex = ex; }

        public String q(String k, String d) { return query.getOrDefault(k, d); }

        @SuppressWarnings("unchecked")
        public Map<String, Object> bodyMap() { return body instanceof Map ? (Map<String, Object>) body : new HashMap<>(); }

        public void json(int code, Object obj) {
            this.code = code;
            try {
                byte[] b = Json.write(obj).getBytes(StandardCharsets.UTF_8);
                ex.getResponseHeaders().set("Content-Type", "application/json");
                ex.getResponseHeaders().set("X-Trace", kit.name);
                ex.getResponseHeaders().set("X-Pod", pod());
                ex.getResponseHeaders().set("X-Powered-By", POWERED_BY);
                ex.getResponseHeaders().set("Access-Control-Allow-Origin", "*");
                ex.getResponseHeaders().set("Access-Control-Expose-Headers", "X-Trace, X-Pod");
                ex.sendResponseHeaders(code, b.length);
                try (OutputStream os = ex.getResponseBody()) { os.write(b); }
            } catch (IOException e) { throw new RuntimeException(e); }
        }

        public void error(int code, String msg) { json(code, Map.of("error", msg, "service", kit.name)); }

        /** Burns roughly WORK_FACTOR milliseconds of CPU to simulate real business logic. */
        public void work() {
            try {
                MessageDigest md = MessageDigest.getInstance("SHA-256");
                byte[] buf = kit.name.getBytes();
                for (int i = 0; i < kit.work * 2500; i++) buf = md.digest(buf);
            } catch (Exception ignored) { }
        }
    }

    public void get(String pattern, Consumer<Ctx> fn) { add("GET", pattern, fn); }
    public void post(String pattern, Consumer<Ctx> fn) { add("POST", pattern, fn); }

    private void add(String method, String pattern, Consumer<Ctx> fn) {
        List<String> names = new ArrayList<>();
        Matcher m = Pattern.compile("\\{(\\w+)}").matcher(pattern);
        while (m.find()) names.add(m.group(1));
        Pattern rx = Pattern.compile("^" + pattern.replaceAll("\\{\\w+}", "([^/]+)") + "$");
        routes.add(new Object[]{method, rx, names, fn});
    }

    private void plain(HttpExchange ex, String text) throws IOException {
        byte[] b = text.getBytes(StandardCharsets.UTF_8);
        ex.getResponseHeaders().set("Content-Type", "text/plain");
        ex.sendResponseHeaders(200, b.length);
        try (OutputStream os = ex.getResponseBody()) { os.write(b); }
    }

    private String metrics() {
        StringBuilder sb = new StringBuilder("# TYPE railshack_http_requests_total counter\n");
        new TreeMap<>(reqs).forEach((k, v) -> {
            String[] p = k.split("\\|");
            sb.append(String.format("railshack_http_requests_total{service=\"%s\",method=\"%s\",code=\"%s\"} %d%n", name, p[0], p[1], v.get()));
        });
        sb.append("# TYPE railshack_http_request_duration_seconds summary\n");
        synchronized (this) {
            sb.append(String.format("railshack_http_request_duration_seconds_sum{service=\"%s\"} %.6f%n", name, durSum));
            sb.append(String.format("railshack_http_request_duration_seconds_count{service=\"%s\"} %d%n", name, durCount));
        }
        sb.append(String.format("railshack_up{service=\"%s\",language=\"java\"} 1%n", name));
        sb.append(String.format("railshack_build_info{service=\"%s\",version=\"%s\",project=\"%s\",source=\"%s\"} 1%n", name, env("SERVICE_VERSION", "1.0.0"), BRAND, SOURCE));
        return sb.toString();
    }

    @SuppressWarnings("unchecked")
    private void handle(HttpExchange ex) throws IOException {
        long start = System.nanoTime();
        String path = ex.getRequestURI().getPath();
        if (path.equals("/healthz")) { plain(ex, "ok"); return; }
        if (path.equals("/readyz")) { plain(ex, "ready"); return; }
        if (path.equals("/metrics")) { plain(ex, metrics()); return; }
        Ctx c = new Ctx(this, ex);
        String rq = ex.getRequestURI().getRawQuery();
        if (rq != null) for (String kv : rq.split("&")) {
            String[] p = kv.split("=", 2);
            c.query.put(URLDecoder.decode(p[0], StandardCharsets.UTF_8), p.length > 1 ? URLDecoder.decode(p[1], StandardCharsets.UTF_8) : "");
        }
        String raw = new String(ex.getRequestBody().readAllBytes(), StandardCharsets.UTF_8);
        if (!raw.isBlank()) { try { c.body = Json.parse(raw); } catch (Exception ignored) { } }
        boolean found = false;
        for (Object[] r : routes) {
            Matcher m = ((Pattern) r[1]).matcher(path);
            if (r[0].equals(ex.getRequestMethod()) && m.matches()) {
                List<String> names = (List<String>) r[2];
                for (int i = 0; i < names.size(); i++) c.params.put(names.get(i), URLDecoder.decode(m.group(i + 1), StandardCharsets.UTF_8));
                found = true;
                try {
                    if (latency > 0) Thread.sleep(latency);
                    if (failRate > 0 && Math.random() < failRate) c.error(500, "injected failure (FAIL_RATE)");
                    else ((Consumer<Ctx>) r[3]).accept(c);
                } catch (Exception e) {
                    c.error(500, e.getClass().getSimpleName() + ": " + e.getMessage());
                }
                break;
            }
        }
        if (!found) c.error(404, "not found");
        double d = (System.nanoTime() - start) / 1e9;
        reqs.computeIfAbsent(ex.getRequestMethod() + "|" + c.code, k -> new AtomicLong()).incrementAndGet();
        synchronized (this) { durSum += d; durCount++; }
        String rid = Optional.ofNullable(ex.getRequestHeaders().getFirst("X-Request-Id")).orElse("");
        System.out.println(String.format("{\"ts\":\"%s\",\"service\":\"%s\",\"method\":\"%s\",\"path\":\"%s\",\"status\":%d,\"ms\":%d,\"requestId\":\"%s\"}",
                Instant.now(), name, ex.getRequestMethod(), path, c.code, (long) (d * 1000), rid));
    }

    public void run() throws IOException {
        int port = intEnv("PORT", 8080);
        HttpServer srv = HttpServer.create(new InetSocketAddress(port), 0);
        srv.createContext("/", ex -> { try { handle(ex); } finally { ex.close(); } });
        srv.setExecutor(Executors.newFixedThreadPool(32));
        Runtime.getRuntime().addShutdownHook(new Thread(() -> srv.stop(2)));
        srv.start();
        System.out.println(String.format("{\"service\":\"%s\",\"msg\":\"listening\",\"port\":%d,\"language\":\"java\",\"project\":\"%s\",\"youtube\":\"%s\"}", name, port, BRAND, YOUTUBE));
    }

    // ---------- minimal JSON ----------
    public static final class Json {
        public static String write(Object o) {
            if (o == null) return "null";
            if (o instanceof String s) return quote(s);
            if (o instanceof Number || o instanceof Boolean) return o.toString();
            if (o instanceof Map<?, ?> m) {
                StringJoiner j = new StringJoiner(",", "{", "}");
                m.forEach((k, v) -> j.add(quote(k.toString()) + ":" + write(v)));
                return j.toString();
            }
            if (o instanceof Collection<?> c) {
                StringJoiner j = new StringJoiner(",", "[", "]");
                c.forEach(v -> j.add(write(v)));
                return j.toString();
            }
            return quote(o.toString());
        }

        static String quote(String s) {
            StringBuilder sb = new StringBuilder("\"");
            for (char ch : s.toCharArray()) {
                switch (ch) {
                    case '"' -> sb.append("\\\"");
                    case '\\' -> sb.append("\\\\");
                    case '\n' -> sb.append("\\n");
                    case '\r' -> sb.append("\\r");
                    case '\t' -> sb.append("\\t");
                    default -> { if (ch < 0x20) sb.append(String.format("\\u%04x", (int) ch)); else sb.append(ch); }
                }
            }
            return sb.append('"').toString();
        }

        public static Object parse(String s) { int[] i = {0}; Object v = val(s, i); return v; }

        static void ws(String s, int[] i) { while (i[0] < s.length() && Character.isWhitespace(s.charAt(i[0]))) i[0]++; }

        static Object val(String s, int[] i) {
            ws(s, i);
            char ch = s.charAt(i[0]);
            if (ch == '{') {
                Map<String, Object> m = new LinkedHashMap<>(); i[0]++; ws(s, i);
                if (s.charAt(i[0]) == '}') { i[0]++; return m; }
                while (true) {
                    ws(s, i); String k = (String) val(s, i); ws(s, i); i[0]++; // ':'
                    m.put(k, val(s, i)); ws(s, i);
                    if (s.charAt(i[0]++) == '}') return m;
                }
            }
            if (ch == '[') {
                List<Object> l = new ArrayList<>(); i[0]++; ws(s, i);
                if (s.charAt(i[0]) == ']') { i[0]++; return l; }
                while (true) { l.add(val(s, i)); ws(s, i); if (s.charAt(i[0]++) == ']') return l; }
            }
            if (ch == '"') {
                StringBuilder sb = new StringBuilder(); i[0]++;
                while (s.charAt(i[0]) != '"') {
                    char c = s.charAt(i[0]++);
                    if (c == '\\') {
                        char e = s.charAt(i[0]++);
                        switch (e) {
                            case 'n' -> sb.append('\n'); case 't' -> sb.append('\t'); case 'r' -> sb.append('\r');
                            case 'u' -> { sb.append((char) Integer.parseInt(s.substring(i[0], i[0] + 4), 16)); i[0] += 4; }
                            default -> sb.append(e);
                        }
                    } else sb.append(c);
                }
                i[0]++; return sb.toString();
            }
            if (s.startsWith("true", i[0])) { i[0] += 4; return true; }
            if (s.startsWith("false", i[0])) { i[0] += 5; return false; }
            if (s.startsWith("null", i[0])) { i[0] += 4; return null; }
            int st = i[0];
            while (i[0] < s.length() && "+-0123456789.eE".indexOf(s.charAt(i[0])) >= 0) i[0]++;
            return Double.parseDouble(s.substring(st, i[0]));
        }
    }
}

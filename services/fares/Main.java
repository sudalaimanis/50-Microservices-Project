// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

import java.util.*;

/** fares — Indian Railways style fare engine: base fare per km, reservation fee, superfast
 *  charge, catering for premium trains, Tatkal surcharge and 5% GST on AC classes. */
public class Main {
    static final Map<String, Double> PER_KM = Map.of("SL", 0.55, "3A", 1.45, "2A", 2.10, "1A", 3.50, "CC", 1.30, "EC", 2.60, "2S", 0.40);
    static final Map<String, Integer> RESERVATION = Map.of("SL", 20, "3A", 40, "2A", 50, "1A", 60, "CC", 40, "EC", 60, "2S", 15);
    static final Map<String, Integer> SUPERFAST = Map.of("SL", 30, "3A", 45, "2A", 45, "1A", 75, "CC", 45, "EC", 75, "2S", 15);
    static final Set<String> AC = Set.of("1A", "2A", "3A", "CC", "EC");
    static final Set<String> PREMIUM = Set.of("Rajdhani", "Shatabdi", "Vande Bharat");

    static double num(Object o, double d) { return o instanceof Number n ? n.doubleValue() : d; }

    static Map<String, Object> quote(String type, double km, String cls, int pax, String quota) {
        double base = Math.max(PER_KM.getOrDefault(cls, 1.0) * km, 60) * (1 + (cls.equals("1A") ? 0.1 : 0));
        double catering = PREMIUM.contains(type) ? Math.round(base * 0.15) : 0;
        double tatkal = "TQ".equals(quota) ? Math.min(Math.max(base * 0.3, 100), 500) : 0;
        double perHead = base + RESERVATION.getOrDefault(cls, 20) + SUPERFAST.getOrDefault(cls, 30) + catering + tatkal;
        double gst = AC.contains(cls) ? perHead * 0.05 : 0;
        double total = Math.round((perHead + gst) * pax * 100) / 100.0;
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("class", cls);
        m.put("baseFare", Math.round(base));
        m.put("reservation", RESERVATION.getOrDefault(cls, 20));
        m.put("superfast", SUPERFAST.getOrDefault(cls, 30));
        m.put("catering", catering);
        m.put("tatkal", Math.round(tatkal));
        m.put("gst", Math.round(gst * 100) / 100.0);
        m.put("perPassenger", Math.round((perHead + gst) * 100) / 100.0);
        m.put("passengers", pax);
        m.put("total", total);
        return m;
    }

    @SuppressWarnings("unchecked")
    public static void main(String[] args) throws Exception {
        Kit s = new Kit("fares");
        s.post("/api/fares/quote", c -> {
            c.work();
            Map<String, Object> b = c.bodyMap();
            String cls = String.valueOf(b.getOrDefault("class", "SL"));
            if (!PER_KM.containsKey(cls)) { c.error(400, "unknown class " + cls); return; }
            c.json(200, quote(String.valueOf(b.get("type")), num(b.get("km"), 100), cls, (int) num(b.get("passengers"), 1), String.valueOf(b.get("quota"))));
        });
        s.post("/api/fares/batch", c -> {
            c.work();
            Map<String, Object> out = new LinkedHashMap<>();
            for (Object o : (List<Object>) c.bodyMap().getOrDefault("items", List.of())) {
                Map<String, Object> it = (Map<String, Object>) o;
                Map<String, Object> perClass = new LinkedHashMap<>();
                for (Object cl : (List<Object>) it.getOrDefault("classes", List.of())) {
                    perClass.put(cl.toString(), quote(String.valueOf(it.get("type")), num(it.get("km"), 100), cl.toString(), 1, "GN").get("total"));
                }
                out.put(String.valueOf(it.get("train")), perClass);
            }
            c.json(200, out);
        });
        s.run();
    }
}

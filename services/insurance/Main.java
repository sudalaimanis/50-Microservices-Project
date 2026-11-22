// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

import java.util.*;

/** insurance — optional travel insurance quote per passenger. */
public class Main {
    public static void main(String[] args) throws Exception {
        Kit s = new Kit("insurance");
        s.get("/api/insurance/quote", c -> {
            int pax = Integer.parseInt(c.q("passengers", "1"));
            Map<String, Object> m = new LinkedHashMap<>();
            m.put("provider", "ShackSure General Insurance");
            m.put("perPassenger", 0.45);
            m.put("premium", Math.round(0.45 * pax * 100) / 100.0);
            m.put("cover", Map.of("death", 1000000, "permanentDisability", 1000000, "hospitalisation", 200000));
            c.json(200, m);
        });
        s.run();
    }
}

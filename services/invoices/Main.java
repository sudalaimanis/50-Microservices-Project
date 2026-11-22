// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

import java.time.LocalDate;
import java.util.*;
import java.util.concurrent.atomic.AtomicLong;

/** invoices — GST tax invoices for every booking. */
public class Main {
    static final AtomicLong SEQ = new AtomicLong(100000);

    static double num(Object o) { return o instanceof Number n ? n.doubleValue() : 0; }

    @SuppressWarnings("unchecked")
    public static void main(String[] args) throws Exception {
        Kit s = new Kit("invoices");
        s.post("/api/invoices/create", c -> {
            Map<String, Object> b = c.bodyMap();
            Map<String, Object> fare = b.get("fare") instanceof Map ? (Map<String, Object>) b.get("fare") : Map.of();
            double gst = num(fare.get("gst")) * num(fare.get("passengers"));
            Map<String, Object> inv = new LinkedHashMap<>();
            inv.put("invoiceNo", "RS/" + LocalDate.now().getYear() + "/" + SEQ.incrementAndGet());
            inv.put("pnr", b.get("pnr"));
            inv.put("billedTo", b.get("user"));
            inv.put("taxable", Math.round((num(b.get("total")) - gst) * 100) / 100.0);
            inv.put("cgst", Math.round(gst / 2 * 100) / 100.0);
            inv.put("sgst", Math.round(gst / 2 * 100) / 100.0);
            inv.put("total", b.get("total"));
            inv.put("gstin", "09AAACR0000R1ZX");
            c.json(201, inv);
        });
        s.run();
    }
}

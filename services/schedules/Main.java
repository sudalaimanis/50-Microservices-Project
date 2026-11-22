// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

import java.time.LocalDate;
import java.util.*;

/** schedules — which days each train runs, and its departure time from origin. */
public class Main {
    // train -> {departs, running days (ISO 1=Mon .. 7=Sun)}
    static final Map<String, String[]> SCHEDULE = new HashMap<>();
    static {
        String[][] rows = {
            {"12301", "16:50", "1234567"}, {"12302", "16:55", "1234567"}, {"12951", "17:00", "1234567"},
            {"12952", "16:55", "1234567"}, {"22436", "06:00", "124567"}, {"22435", "15:00", "124567"},
            {"12560", "20:05", "1234567"}, {"12559", "19:30", "1234567"}, {"12002", "06:00", "1234567"},
            {"12627", "19:20", "1234567"}, {"12839", "23:45", "1234567"}, {"12123", "17:10", "1234567"},
            {"12957", "17:40", "1234567"}, {"12013", "16:30", "1234567"}, {"12015", "06:10", "123456"},
            {"22691", "20:00", "1234567"}, {"12391", "07:30", "1234567"}, {"12230", "22:00", "1234567"}};
        for (String[] r : rows) SCHEDULE.put(r[0], new String[]{r[1], r[2]});
    }

    static String dayNames(String d) {
        String[] n = {"", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"};
        if (d.length() == 7) return "Daily";
        StringJoiner j = new StringJoiner(" ");
        for (char ch : d.toCharArray()) j.add(n[ch - '0']);
        return j.toString();
    }

    public static void main(String[] args) throws Exception {
        Kit s = new Kit("schedules");
        s.get("/api/schedules/runs", c -> {
            LocalDate date;
            try { date = LocalDate.parse(c.q("date", "")); } catch (Exception e) { c.error(400, "date must be YYYY-MM-DD"); return; }
            String dow = String.valueOf(date.getDayOfWeek().getValue());
            Map<String, Object> out = new LinkedHashMap<>();
            for (String no : c.q("numbers", "").split(",")) {
                String[] sch = SCHEDULE.get(no);
                if (sch == null) continue;
                out.put(no, Map.of("runs", sch[1].contains(dow), "days", dayNames(sch[1]), "departs", sch[0]));
            }
            c.json(200, out);
        });
        s.get("/api/schedules/{train}", c -> {
            String[] sch = SCHEDULE.get(c.params.get("train"));
            if (sch == null) { c.error(404, "unknown train"); return; }
            c.json(200, Map.of("train", c.params.get("train"), "departs", sch[0], "days", dayNames(sch[1])));
        });
        s.run();
    }
}

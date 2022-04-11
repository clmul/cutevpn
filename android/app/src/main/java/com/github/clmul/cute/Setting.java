package com.github.clmul.cute;

import android.app.Activity;
import android.content.SharedPreferences;
import android.content.pm.ApplicationInfo;
import android.content.pm.PackageManager;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

class Setting {
    private Activity activity;
    private SharedPreferences pref;
    static final String name = "Name";
    static final String ip = "IP Address";
    static final String gateway = "Gateway";
    static final String dns = "DNS Server";
    static final String excludedApps = "Excluded Apps";
    static final String links = "Links";
    static final String logViewer = "Log Viewer";

    Setting(Activity activity) {
        this.activity = activity;
        this.pref = activity.getPreferences(Activity.MODE_PRIVATE);
        setDefault();
    }

    private void setDefault() {
        if (getExcludedApps() == null) {
            saveExcludedApps(new HashSet<>());
        }
        if (getString(dns) == null) {
            saveString(dns, "8.8.8.8");
        }
        if (getString(name) == null) {
            saveString(name, "cellphone");
        }
        if (getString(ip) == null) {
            saveString(ip, "172.20.0.99");
        }
        if (getString(gateway) == null) {
            saveString(gateway, "172.20.0.1");
        }
        if (getString(links) == null) {
            saveString(links, "tls://hostname:4433");
        }
    }

    private Map<String, String> item(String key, boolean readValue) {
        // generate a map for the setting item in the list.
        // if readValue is true, the value will show in the list.
        HashMap<String, String> r = new HashMap<>();
        r.put("name", key);
        if (readValue) {
            r.put("value", getString(key));
        } else {
            r.put("valud", "");
        }
        return r;
    }

    List<Map<String, String>> getSettingList() {
        List<Map<String, String>> items = new ArrayList<>();
        items.add(item(name, true));
        items.add(item(ip, true));
        items.add(item(gateway, true));
        items.add(item(links, false));
        items.add(item(dns, true));
        items.add(item(excludedApps, false));
        items.add(item(logViewer, false));
        return items;
    }

    String getString(String key) {
        return pref.getString(key, null);
    }

    void saveString(String k, String v) {
        pref.edit().putString(k, v).apply();
    }

    void saveExcludedApps(Set<String> apps) {
        pref.edit().putStringSet(excludedApps, apps).apply();
    }

    Set<String> getExcludedApps() {
        Set<String> apps = pref.getStringSet(excludedApps, null);
        if (apps == null) {
            return null;
        }
        return new HashSet<>(apps);
    }

    class Application {
        String name;
        String packageName;

        Application(String name, String packageName) {
            this.name = name;
            this.packageName = packageName;
        }
    }

    List<Application> getAllApps() {
        final PackageManager pm = activity.getPackageManager();
        List<ApplicationInfo> packages = pm.getInstalledApplications(PackageManager.GET_META_DATA);
        List<Application> result = new ArrayList<>();
        for (ApplicationInfo info : packages) {
            if ((info.flags & ApplicationInfo.FLAG_SYSTEM) != 0) {
                continue;
            }
            String appName = pm.getApplicationLabel(info).toString();
            String packageName = info.packageName;
            Application app = new Application(appName, packageName);
            result.add(app);
        }
        return result;
    }
}

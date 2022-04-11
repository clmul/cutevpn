package com.github.clmul.cute;

import android.content.Intent;
import android.content.pm.PackageManager;
import android.net.VpnService;
import android.os.Binder;
import android.os.IBinder;
import android.os.ParcelFileDescriptor;
import android.util.Log;

import java.util.Set;

import cutevpn.Cutevpn;
import cutevpn.Neighbors;
import cutevpn.VPN;

public class VPNService extends VpnService {
    private final static String TAG = "VPNService";
    private final IBinder binder = new LocalBinder();
    private VPN vpn;

    class LocalBinder extends Binder {
        VPNService getService() {
            return VPNService.this;
        }
    }

    @Override
    public IBinder onBind(Intent intent) {
        return binder;
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        Log.w(TAG, "onStartCommand");
        return START_NOT_STICKY;
    }

    public String start(Setting setting) {
        Builder builder = new Builder();

        String ip = setting.getString(Setting.ip);
        builder.addAddress(ip, 24);
        Set<String> excluded = setting.getExcludedApps();
        excluded.add(getPackageName());
        for (String packageName : excluded) {
            try {
                builder.addDisallowedApplication(packageName);
            } catch (PackageManager.NameNotFoundException e) {
                Log.d(TAG, e.getMessage());
            }
        }
        builder.allowBypass();
        builder.addDnsServer(setting.getString(Setting.dns));
        builder.addRoute("0.0.0.0", 0);
        builder.setMtu(1350);
        builder.setSession("CuteVPN");
        ParcelFileDescriptor pfd = builder.establish();
        int fd = pfd.detachFd();

        String dir = getFilesDir().getPath();
        String gateway = setting.getString(Setting.gateway);
        String name = setting.getString(Setting.name);
        String links = setting.getString(Setting.links);
        vpn = Cutevpn.setup(fd, dir, name, ip, gateway, links);
        try {
            vpn.start();
        } catch (Exception e) {
            return e.getMessage();
        }
        return "";
    }

    public boolean isRunning() {
        return vpn != null && vpn.isRunning();
    }

    public void updateGateway(String gateway) {
        vpn.updateGateway(gateway);
    }

    public Neighbors getNeighbors() {
        return vpn.getNeighbors();
    }

    public void stop() {
        Log.w(TAG, "stop");
        vpn.stop();
    }
}

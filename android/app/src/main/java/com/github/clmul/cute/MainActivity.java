package com.github.clmul.cute;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.ComponentName;
import android.content.Context;
import android.content.DialogInterface;
import android.content.Intent;
import android.content.ServiceConnection;
import android.os.Bundle;
import android.os.IBinder;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.view.View;
import android.widget.AdapterView;
import android.widget.Button;
import android.widget.EditText;
import android.widget.ListAdapter;
import android.widget.ListView;
import android.widget.SimpleAdapter;
import android.widget.Toast;

import java.util.List;
import java.util.Map;
import java.util.Set;

import cutevpn.Neighbors;

public class MainActivity extends Activity {
    private static final String TAG = "MainActivity";
    private static final int reqCode = 1911;
    private Setting setting;

    private ListView settings;
    private VPNService service;
    private Button start, stop;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setting = new Setting(this);

        setContentView(R.layout.activity_main);

        start = findViewById(R.id.start);
        stop = findViewById(R.id.stop);
        settings = findViewById(R.id.settings);
        settings.addHeaderView(new View(this));
        settings.addFooterView(new View(this));

        settings.setOnItemClickListener((AdapterView<?> parent, View view, int position, long id) -> {
            Object item = parent.getItemAtPosition(position);
            String name = (String) ((Map) item).get("name");
            switch (name) {
                case Setting.excludedApps:
                    openExcludedAppDialog();
                    break;
                case Setting.name:
                case Setting.ip:
                case Setting.dns:
                    openStringSettingDialog(name);
                    break;
                case Setting.links:
                    openTextSettingDialog(name);
                    break;
                case Setting.gateway:
                    if (stop.isEnabled()) {
                        openGatewayListDialog();
                    }
                    if (start.isEnabled()) {
                        openStringSettingDialog(name);
                    }
                    break;
                case Setting.logViewer:
                    Intent intent = new Intent(this, LogViewer.class);
                    startActivity(intent);
                    break;
            }

        });
        updateSettings();

        Intent intent = new Intent(this, VPNService.class);
        startService(intent);
        bindService(intent, connection, Context.BIND_AUTO_CREATE);

        start.setOnClickListener((View view) -> {
            start.setEnabled(false);
            Intent prompt = VPNService.prepare(this);
            if (prompt != null) {
                startActivityForResult(prompt, reqCode);
                return;
            }
            startVPN();
        });

        stop.setOnClickListener((View view) -> {
            stop.setEnabled(false);
            start.setEnabled(true);
            service.stop();
        });
    }

    private void updateSettings() {
        ListAdapter settingsAdapter = new SimpleAdapter(this, setting.getSettingList(),
                R.layout.setting_item, new String[]{"name", "value"},
                new int[]{R.id.setting_item_name, R.id.setting_item_value});
        settings.setAdapter(settingsAdapter);
    }

    private void openStringSettingDialog(String name) {
        String v = setting.getString(name);
        EditText input = (EditText) getLayoutInflater().inflate(R.layout.string_setting, null);
        input.setText(v);
        new AlertDialog.Builder(this).setTitle(name)
                .setPositiveButton("OK", (DialogInterface dialog, int buttonID) -> {
                    String newValue = input.getText().toString();
                    setting.saveString(name, newValue);
                    updateSettings();
                })
                .setView(input)
                .setNegativeButton("Cancel", null)
                .create().show();
    }

    private void openTextSettingDialog(String name) {
        String v = setting.getString(name);
        EditText input = (EditText) getLayoutInflater().inflate(R.layout.text_setting, null);
        input.setText(v);
        new AlertDialog.Builder(this).setTitle(name)
                .setPositiveButton("OK", (DialogInterface dialog, int buttonID) -> {
                    String newValue = input.getText().toString();
                    setting.saveString(name, newValue);
                    updateSettings();
                })
                .setView(input)
                .setNegativeButton("Cancel", null)
                .create().show();
    }

    private void openGatewayListDialog() {
        Neighbors neighbors = service.getNeighbors();

        CharSequence[] neighborsText = new CharSequence[neighbors.n()];

        for (int i = 0; i < neighbors.n(); i++) {
            String addr = neighbors.addr(i);
            String name = neighbors.name(i);
            neighborsText[i] = name + "  " + addr;
        }

        new AlertDialog.Builder(this).setTitle(Setting.gateway)
                .setItems(neighborsText, (DialogInterface dialog, int which) -> {
                    String gateway = neighbors.addr(which);
                    setting.saveString(Setting.gateway, gateway);
                    service.updateGateway(gateway);
                    updateSettings();
                })
                .create().show();
    }

    private void openExcludedAppDialog() {
        List<Setting.Application> allApps = setting.getAllApps();
        Set<String> excluded = setting.getExcludedApps();

        allApps.sort((Setting.Application a, Setting.Application b) -> {
            String p1 = a.packageName;
            String p2 = b.packageName;
            if (excluded.contains(p1) && !excluded.contains(p2)) {
                return -1;
            }
            if (!excluded.contains(p1) && excluded.contains(p2)) {
                return 1;
            }
            return a.name.compareTo(b.name);
        });

        CharSequence[] apps = new CharSequence[allApps.size()];
        boolean[] checked = new boolean[allApps.size()];

        for (int i = 0; i < allApps.size(); i++) {
            Setting.Application app = allApps.get(i);
            apps[i] = app.name;
            checked[i] = excluded.contains(app.packageName);
        }

        new AlertDialog.Builder(this).setTitle(Setting.excludedApps)
                .setPositiveButton("OK", (DialogInterface dialog, int buttonID) -> {
                    setting.saveExcludedApps(excluded);
                })
                .setNegativeButton("Cancel", null)
                .setMultiChoiceItems(apps, checked, (DialogInterface dialog, int which, boolean isChecked) -> {
                    if (isChecked) {
                        excluded.add(allApps.get(which).packageName);
                    } else {
                        excluded.remove(allApps.get(which).packageName);
                    }
                }).create().show();
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        MenuItem item = menu.add("add");
        item.setIcon(R.drawable.ic_add);
        item.setShowAsAction(MenuItem.SHOW_AS_ACTION_ALWAYS);
        return super.onCreateOptionsMenu(menu);
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        String title = item.getTitle().toString();
        if (title.equals("add")) {
            Log.d(TAG, "add config");
            return true;
        }
        return super.onOptionsItemSelected(item);
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        if (requestCode == reqCode && resultCode == RESULT_OK) {
            startVPN();
        }
    }

    void startVPN() {
        Activity activity = this;
        new Thread(() -> {
            String result = service.start(setting);
            runOnUiThread(()->{
                if (!result.equals("")) {
                    Log.e(TAG, result);
                    start.setEnabled(true);

                    Toast.makeText(activity, result, Toast.LENGTH_LONG).show();
                    return;
                }
                stop.setEnabled(true);
            });
        }).start();
    }

    @Override
    public void onDestroy() {
        unbindService(connection);
        super.onDestroy();
    }

    private ServiceConnection connection = new ServiceConnection() {
        @Override
        public void onServiceConnected(ComponentName className, IBinder binder) {
            service = ((VPNService.LocalBinder) binder).getService();
            if (service.isRunning()) {
                stop.setEnabled(true);
            } else {
                start.setEnabled(true);
            }
        }

        @Override
        public void onServiceDisconnected(ComponentName arg0) {
        }
    };
}

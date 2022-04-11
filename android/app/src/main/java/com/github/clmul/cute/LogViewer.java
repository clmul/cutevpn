package com.github.clmul.cute;

import android.app.Activity;
import android.os.Bundle;
import android.util.Log;
import android.widget.TextView;


import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Arrays;


public class LogViewer extends Activity {

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_log_viewer);
        new Thread(() -> {
            File dir = getFilesDir();
            File[] files = dir.listFiles();
            Arrays.sort(files, (a, b) -> b.getName().compareTo(a.getName()));
            Path filename = Paths.get(dir.getPath(), files[0].getName());

            String content;
            try {
                content = new String(Files.readAllBytes(filename));
            } catch (IOException e) {
                content = "";
            }
            String finalContent = content;
            runOnUiThread(() -> {
                TextView log = findViewById(R.id.log_content);
                log.setText(finalContent);
            });
        }).start();
    }
}

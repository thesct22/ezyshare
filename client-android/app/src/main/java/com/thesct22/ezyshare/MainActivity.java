package com.thesct22.ezyshare;

import androidx.appcompat.app.AppCompatActivity;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.content.Intent;
import android.os.Bundle;
import android.text.Editable;
import android.text.TextWatcher;
import android.view.View;
import android.widget.Button;
import android.widget.EditText;
import android.widget.TextView;

import java.security.SecureRandom;

public class MainActivity extends AppCompatActivity {

    private static final String AB = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";
    private static SecureRandom rnd = new SecureRandom();

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        Button uniqueIDButton= findViewById(R.id.uniqueIDButton);
        Button sendLinkButtom= findViewById(R.id.sendLinkButton);
        Button receiveLinkButton= findViewById(R.id.receiveLinkButton);

        EditText uniqueID= findViewById(R.id.uniqueID);
        EditText sendLink= findViewById(R.id.sendLink);
        EditText receiveLink= findViewById(R.id.receiveLink);

        uniqueIDButton.setOnClickListener(new View.OnClickListener() {
            public void onClick(View v) {
                String id=randomString();
                uniqueID.setText(id);
            }
        });

        ClipboardManager clipboard = (ClipboardManager) getSystemService(Context.CLIPBOARD_SERVICE);

        uniqueID.addTextChangedListener(new TextWatcher() {
            @Override
            public void beforeTextChanged(CharSequence s, int start, int count, int after) {

            }

            @Override
            public void onTextChanged(CharSequence s, int start, int before, int count) {
                sendLink.setText("https://thesct22-ezyshare.herokuapp.com/send/"+s);
                receiveLink.setText("https://thesct22-ezyshare.herokuapp.com/recv/"+s);
                EzyShare.setID(s.toString());
            }

            @Override
            public void afterTextChanged(Editable s) {

            }
        });
        sendLink.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                ClipData clip = ClipData.newPlainText("Copied Text", sendLink.getText());
                clipboard.setPrimaryClip(clip);
            }
        });

        receiveLink.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                ClipData clip = ClipData.newPlainText("Copied Text", receiveLink.getText());
                clipboard.setPrimaryClip(clip);
            }
        });
    }

    private String randomString(){
        StringBuilder sb = new StringBuilder(8);
        for(int i = 0; i < 8; i++)
            sb.append(AB.charAt(rnd.nextInt(AB.length())));
        return sb.toString();
    }

    public void sendFile(View view) {
        Intent intent = new Intent(MainActivity.this, SendActivity.class);
        startActivity(intent);
    }

    public void receiveFile(View view) {
        Intent intent = new Intent(MainActivity.this, ReceiveActivity.class);
        startActivity(intent);
    }
}
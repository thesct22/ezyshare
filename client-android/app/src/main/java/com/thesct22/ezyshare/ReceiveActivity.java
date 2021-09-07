package com.thesct22.ezyshare;

import androidx.appcompat.app.AppCompatActivity;

import android.os.Bundle;
import android.util.Log;
import android.widget.Toast;

import com.neovisionaries.ws.client.HostnameUnverifiedException;
import com.neovisionaries.ws.client.OpeningHandshakeException;
import com.neovisionaries.ws.client.WebSocket;
import com.neovisionaries.ws.client.WebSocketAdapter;
import com.neovisionaries.ws.client.WebSocketException;
import com.neovisionaries.ws.client.WebSocketFactory;
import com.neovisionaries.ws.client.WebSocketListener;

import java.io.IOException;

public class ReceiveActivity extends AppCompatActivity {

    private static final String TAG = "ReceiveActivity";
    WebSocket ws=null;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_receive);
        //Secrets is a java class I made which contains the URL to the websocket server. Make a class and put one for yourself

        //Make this Server.java

        //        public class Secrets {
        //            private static String serverWS= "wss://your_server_url.com;
        //            private static String serverURL= "https://your_server_url.com";
        //
        //            public static String getServerWS(){
        //                return serverWS;
        //            }
        //            public static String getServerURL(){
        //                return serverURL;
        //            }
        //        }

        String serverWS= Secrets.getServerWS();
        String ID= EzyShare.getID();
        Log.e(TAG, "hello");


        WebSocketFactory factory = new WebSocketFactory().setConnectionTimeout(5000);

        // Create a WebSocket. The timeout value set above is used.
        try {
            Log.e(TAG,serverWS+"/recv?roomID="+ID);
            ws = factory.createSocket(serverWS+"/recv?roomID="+ID);

            ws.addListener(new WebSocketAdapter() {
                @Override
                public void onTextMessage(WebSocket websocket, String message) throws Exception {
                    Log.d("TAG", "onTextMessage: " + message);
                }
            });

            ws.connectAsynchronously();
        } catch (IOException e) {
            e.printStackTrace();
            Log.e(TAG, e.toString());

        }








//        try {
//            ws = new WebSocketFactory().createSocket(serverWS+"/recv/"+ID,5000);
//            Log.e(TAG, "hi");
//
//            try
//            {
//                // Connect to the server and perform an opening handshake.
//                // This method blocks until the opening handshake is finished.
//                ws.connectAsynchronously();
//
//                ws.addListener(new WebSocketAdapter() {
//                    @Override
//                    public void onTextMessage(WebSocket websocket, String message) throws Exception {
//                        // Received a text message.
//                        Log.e(TAG, "hola");
//                        Toast.makeText(getApplicationContext(), message, Toast.LENGTH_LONG).show();
//                        Log.e(TAG, message);
//                        Log.e(TAG, websocket.toString());
//
//                    }
//                });
//            }
//            catch (Exception e)
//            {
//                Log.e(TAG,e.toString());
//                // A violation against the WebSocket protocol was detected
//                // during the opening handshake.
//            }
//
//
//            ws.addListener(new WebSocketAdapter() {
//                @Override
//                public void onTextMessage(WebSocket websocket, String message) throws Exception {
//                    // Received a text message.
//                    Log.e(TAG, "hola");
//                    Toast.makeText(getApplicationContext(), message, Toast.LENGTH_LONG).show();
//                    Log.e(TAG, message);
//                    Log.e(TAG, websocket.toString());
//
//                }
//            });
//        } catch (IOException e) {
//            e.printStackTrace();
//            Toast.makeText( getApplicationContext(), e.toString(), Toast.LENGTH_SHORT).show();
//            Log.e(TAG, e.toString());
//        }
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();

        if (ws != null) {
            ws.disconnect();
            ws = null;
        }
    }
}
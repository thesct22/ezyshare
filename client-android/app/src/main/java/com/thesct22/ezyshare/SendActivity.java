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
import com.neovisionaries.ws.client.WebSocketFrame;

import java.io.IOException;

public class SendActivity extends AppCompatActivity {
    private static final String TAG = "SendActivity";
    WebSocket ws=null;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_send);


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





        WebSocketFactory factory = new WebSocketFactory().setConnectionTimeout(5000);

        // Create a WebSocket. The timeout value set above is used.
        try {
            ws = factory.createSocket(serverWS+"/send?roomID="+ID);
            Log.e(TAG,serverWS+"/send?roomID="+ID);

            ws.addListener(new WebSocketAdapter() {
                @Override
                public void onTextMessage(WebSocket websocket, String message) throws Exception {
                    Log.d("TAG", "onTextMessage: " + message);
                }

            });

            ws.connectAsynchronously();
        } catch (Exception e) {
            e.printStackTrace();
            Log.e(TAG, e.toString());
        }









//        try {
//            WebSocket ws = new WebSocketFactory().createSocket(serverWS+"/send/"+ID,5000);
//            Log.e(TAG, "hi");
//            Log.e(TAG, ws.toString());
//
//            try
//            {
//                // Connect to the server and perform an opening handshake.
//                // This method blocks until the opening handshake is finished.
//                ws.connectAsynchronously();
//                ws.addListener(new WebSocketAdapter() {
//                    @Override
//                    public void onTextMessage(WebSocket websocket, String message) throws Exception {
//                        // Received a text message.
//                        Log.e(TAG, "hola");
//                        Toast.makeText(getApplicationContext(), message, Toast.LENGTH_LONG).show();
//                        Log.e(TAG, message);
//                        Log.e(TAG, websocket.toString());
//
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
//        } catch (IOException e) {
//            e.printStackTrace();
//            Toast.makeText( getApplicationContext(), e.toString(), Toast.LENGTH_SHORT).show();
//            Log.e(TAG, e.toString());
//
//        }

    }
}
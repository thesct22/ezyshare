package com.thesct22.ezyshare;

import android.app.Application;

public class EzyShare extends Application {
    private static String ID;
    public static String getID(){
        return ID;
    }
    public static void setID(String s){
        ID=s;
    }

}

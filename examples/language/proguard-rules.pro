# Add project specific ProGuard rules here.
# You can control the set of applied configuration files using the
# proguardFiles setting in build.gradle.

# Keep the line number information for debugging stack traces.
-keepattributes SourceFile,LineNumberTable

-keep class com.example.app.model.** { *; }
-keepclassmembers class * implements android.os.Parcelable {
    public static final ** CREATOR;
}
-dontwarn okio.**

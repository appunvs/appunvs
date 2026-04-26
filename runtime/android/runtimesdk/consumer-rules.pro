# Consumer ProGuard rules — applied to every host that links this AAR.
# D2.a empty shell has no JNI / reflection / serialization, so nothing
# to keep yet.  PR D2.c will add `-keep class com.appunvs.runtimesdk.**`
# rules for the JNI bridge surface.

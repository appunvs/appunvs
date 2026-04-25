// JsonConverterFactory — a tiny Retrofit Converter.Factory that uses
// kotlinx.serialization to encode request bodies and decode response
// bodies as JSON.
//
// We hand-roll this rather than depend on a community port because:
//   (a) the community ports drift between Kotlin/serialization releases
//       (com.jakewharton.retrofit:retrofit2-kotlinx-serialization-converter
//       was last published in 2021)
//   (b) the surface we need is ~30 lines and well-defined
//   (c) it lets us pin our own Json instance behaviour (ignoreUnknownKeys
//       so the relay can add fields without breaking old clients)
package com.appunvs.runtime.net

import kotlinx.serialization.KSerializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.serializer
import okhttp3.MediaType
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.ResponseBody
import retrofit2.Converter
import retrofit2.Retrofit
import java.lang.reflect.Type

class JsonConverterFactory(
    private val json: Json,
    private val mediaType: MediaType = "application/json".toMediaType(),
) : Converter.Factory() {

    override fun responseBodyConverter(
        type: Type,
        annotations: Array<out Annotation>,
        retrofit: Retrofit,
    ): Converter<ResponseBody, *> {
        @Suppress("UNCHECKED_CAST")
        val serializer = json.serializersModule.serializer(type) as KSerializer<Any?>
        return Converter<ResponseBody, Any?> { body ->
            val text = body.string()
            // Retrofit suspend functions with no return type produce an
            // empty Unit body — short-circuit decoding.
            if (text.isEmpty() && type === Unit::class.java) {
                Unit
            } else {
                json.decodeFromString(serializer, text)
            }
        }
    }

    override fun requestBodyConverter(
        type: Type,
        parameterAnnotations: Array<out Annotation>,
        methodAnnotations: Array<out Annotation>,
        retrofit: Retrofit,
    ): Converter<*, RequestBody> {
        @Suppress("UNCHECKED_CAST")
        val serializer = json.serializersModule.serializer(type) as KSerializer<Any?>
        return Converter<Any?, RequestBody> { value ->
            json.encodeToString(serializer, value).toRequestBody(mediaType)
        }
    }
}

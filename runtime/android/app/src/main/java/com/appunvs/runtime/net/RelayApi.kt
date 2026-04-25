// RelayApi — Retrofit interface for the relay's REST surface.
//
// /ai/turn (SSE) is intentionally NOT here — Retrofit doesn't stream
// well; we issue that one via OkHttp directly in AISSEClient.
package com.appunvs.runtime.net

import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path

interface RelayApi {

    // Auth: signup / login are public; register / me require a session
    // token in the Authorization header (added by the AuthInterceptor).
    @POST("/auth/signup")
    suspend fun signup(@Body body: SignupRequest): SessionResponse

    @POST("/auth/login")
    suspend fun login(@Body body: LoginRequest): SessionResponse

    @POST("/auth/register")
    suspend fun registerDevice(@Body body: RegisterRequest): RegisterResponse

    @GET("/auth/me")
    suspend fun me(): MeResponse

    // Box: device-token authenticated.
    @GET("/box")
    suspend fun listBoxes(): BoxListResponse

    @POST("/box")
    suspend fun createBox(@Body body: BoxCreateRequest): BoxResponse

    @GET("/box/{id}")
    suspend fun getBox(@Path("id") id: String): BoxResponse

    @DELETE("/box/{id}")
    suspend fun archiveBox(@Path("id") id: String)

    // Pair: device-token authenticated.
    @POST("/pair")
    suspend fun pairIssue(@Body body: PairRequestBody): PairResponse

    @POST("/pair/{code}/claim")
    suspend fun pairClaim(
        @Path("code") code: String,
        @Body body: PairClaimRequest,
    ): PairClaimResponse
}

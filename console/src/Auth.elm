module Auth exposing
    ( Model
    , Msg
    , Token
    , UserInfo
    , init
    , signedRequest
    , tokenRequest
    , update
    , userInfoRequest
    )

import Http
import Json.Decode as Decoder exposing (Decoder)
import Task
import Url.Builder as UrlBuilder


type alias Token =
    { idToken : String
    , accessToken : String
    , refreshToken : String
    }


type alias Model =
    { clientId : String
    , tokenEndpoint : String
    , userInfoEndpoint : String
    , redirectUri : String
    , tokens : Maybe Token
    }


init :
    Maybe
        { idToken : String
        , accessToken : String
        , refreshToken : String
        }
    -> String
    -> String
    -> String
    -> Model
init initToken clientId idp redirectUri =
    { clientId = clientId
    , tokenEndpoint = UrlBuilder.crossOrigin idp [ "oauth2", "token" ] []
    , userInfoEndpoint = UrlBuilder.crossOrigin idp [ "oauth2", "userInfo" ] []
    , redirectUri = redirectUri
    , tokens =
        Maybe.map
            (\t ->
                { idToken = t.idToken
                , accessToken = t.accessToken
                , refreshToken = t.refreshToken
                }
            )
            initToken
    }


type alias AuthToken =
    { idToken : String
    , accessToken : String
    , refreshToken : Maybe String
    , expiresIn : Int
    , tokenType : String
    }


authTokenDecoder : Decoder AuthToken
authTokenDecoder =
    Decoder.map5 AuthToken
        (Decoder.field "id_token" Decoder.string)
        (Decoder.field "access_token" Decoder.string)
        (Decoder.maybe <| Decoder.field "refresh_token" Decoder.string)
        (Decoder.field "expires_in" Decoder.int)
        (Decoder.field "token_type" Decoder.string)


toFormParam : List ( String, String ) -> String
toFormParam =
    String.join "&" << List.map (\( a, b ) -> String.join "=" [ a, b ])


tokenRequest :
    msg
    -> (Msg -> msg)
    -> Model
    -> Maybe String
    -> Cmd msg
tokenRequest redirectToLoginForm toMsg model maybeCode =
    let
        refreshTokenRequest tokens =
            Http.post
                { url = model.tokenEndpoint
                , body =
                    Http.stringBody "application/x-www-form-urlencoded" <|
                        toFormParam <|
                            [ ( "grant_type", "refresh_token" )
                            , ( "refresh_token", tokens.refreshToken )
                            , ( "client_id", model.clientId )
                            , ( "redirect_uri", model.redirectUri )
                            ]
                , expect = Http.expectJson (toMsg << AuthTokenResponse) authTokenDecoder
                }

        authorizationCodeRequest code =
            Http.post
                { url = model.tokenEndpoint
                , body =
                    Http.stringBody "application/x-www-form-urlencoded" <|
                        toFormParam <|
                            [ ( "grant_type", "authorization_code" )
                            , ( "code", code )
                            , ( "client_id", model.clientId )
                            , ( "redirect_uri", model.redirectUri )
                            ]
                , expect = Http.expectJson (toMsg << AuthTokenResponse) authTokenDecoder
                }
    in
    case ( model.tokens, maybeCode ) of
        ( Just tokens, _ ) ->
            if tokens.refreshToken /= "" then
                refreshTokenRequest tokens

            else
                case maybeCode of
                    Just code ->
                        authorizationCodeRequest code

                    Nothing ->
                        Task.perform (always redirectToLoginForm) <| Task.succeed ()

        ( Nothing, Just code ) ->
            authorizationCodeRequest code

        ( Nothing, Nothing ) ->
            Task.perform (always redirectToLoginForm) <| Task.succeed ()


type Msg
    = AuthTokenResponse (Result Http.Error AuthToken)


maybe : b -> (a -> b) -> Maybe a -> b
maybe b f =
    Maybe.withDefault b << Maybe.map f


signedRequest :
    Model
    ->
        { method : String
        , headers : List Http.Header
        , url : String
        , body : Http.Body
        , expect : Http.Expect msg
        }
    -> Cmd msg
signedRequest model req =
    let
        headers =
            maybe req.headers
                (\tokens ->
                    Http.header "Authorization" ("Bearer " ++ tokens.accessToken) :: req.headers
                )
                model.tokens
    in
    Http.request
        { method = req.method
        , headers = headers
        , url = req.url
        , body = req.body
        , expect = req.expect
        , timeout = Nothing
        , tracker = Nothing
        }


update :
    (Result Http.Error Token -> msg)
    -> Msg
    -> Model
    -> ( Model, Cmd msg )
update authResult msg model =
    case msg of
        AuthTokenResponse result ->
            case result of
                Ok authToken ->
                    let
                        tokens =
                            { idToken = authToken.idToken
                            , accessToken = authToken.accessToken
                            , refreshToken =
                                case ( authToken.refreshToken, model.tokens ) of
                                    ( Just refreshToken, _ ) ->
                                        refreshToken

                                    ( Nothing, Just tokens_ ) ->
                                        tokens_.refreshToken

                                    ( Nothing, Nothing ) ->
                                        ""
                            }
                    in
                    ( { model | tokens = Just tokens }
                    , Task.perform authResult <| Task.succeed <| Ok tokens
                    )

                Err err ->
                    ( { model | tokens = Nothing }
                    , Task.perform authResult <| Task.succeed <| Err err
                    )


type alias UserInfo =
    { sub : String
    , name : Maybe String
    , email : Maybe String
    }


userInfoDecoder : Decoder UserInfo
userInfoDecoder =
    Decoder.map3 UserInfo
        (Decoder.field "sub" Decoder.string)
        (Decoder.maybe <| Decoder.field "name" Decoder.string)
        (Decoder.maybe <| Decoder.field "email" Decoder.string)


userInfoRequest :
    (Result Http.Error UserInfo -> msg)
    -> Model
    -> Cmd msg
userInfoRequest userInfoMsg model =
    case model.tokens of
        Just tokens ->
            Http.request
                { method = "POST"
                , headers = [ Http.header "Authorization" <| "Bearer " ++ tokens.accessToken ]
                , url = model.userInfoEndpoint
                , body = Http.emptyBody
                , expect = Http.expectJson userInfoMsg userInfoDecoder
                , timeout = Nothing
                , tracker = Nothing
                }

        Nothing ->
            -- TODO
            Cmd.none

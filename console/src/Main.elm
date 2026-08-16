port module Main exposing (main)

import Api
import Auth
import Browser
import Browser.Events as Events
import Browser.Navigation as Nav
import Bytes exposing (Bytes)
import Element exposing (Element)
import Element.Lazy as Lazy
import Http
import Proto.Api as PB
import Route exposing (Route)
import Task
import Url exposing (Url)
import Url.Builder as UrlBuilder
import View.FileUpload
import View.Files
import View.Index
import View.InitUser
import View.Org.Header


port storeTokens : ( String, String, String ) -> Cmd msg


port removeTokens : () -> Cmd msg


type alias Flags =
    { idToken : Maybe String
    , accessToken : Maybe String
    , refreshToken : Maybe String
    , windowWidth : Int
    , windowHeight : Int
    , authClientId : String
    , authRedirectURL : String
    , logoutRedirectURL : String
    , idp : String
    }


type Msg
    = NOP
    | UrlRequest Browser.UrlRequest
    | UrlChanged Url
    | OnResize Int Int
    | AuthResult Msg (Result Http.Error Auth.Token)
    | RedirectToLoginForm
    | ChangeRoute Route
    | AuthMsg Msg Auth.Msg
    | ApiResponse Api.Request (Result Http.Error Api.Response)
    | RetryApiRequest Api.Request
    | FileSelected String
    | FileDelete String Int
    | FileUploadRequested
    | FileUploadMsg View.FileUpload.Msg
    | HeaderMsg View.Org.Header.Msg
    | InitUserMsg View.InitUser.Msg
    | InitUserCommit PB.InitUserRequest
    | InitUserCancel


type alias Token =
    { idToken : String
    , accessToken : String
    , refreshToken : String
    }


type alias Model =
    { key : Nav.Key
    , route : Route
    , windowSize : ( Int, Int )
    , loginFormURL : String
    , logoutURL : String
    , logoutRedirectURL : String
    , authModel : Auth.Model
    , endpoint : String
    , filesModel : View.Files.Model
    , fileUploadModel : View.FileUpload.Model
    , headerModel : View.Org.Header.Model
    , initUserModel : View.InitUser.Model
    }


isJust : Maybe a -> Bool
isJust m =
    case m of
        Just _ ->
            True

        Nothing ->
            False


allJust : List (Maybe a) -> Bool
allJust ms =
    case ms of
        x :: xs ->
            if isJust x then
                allJust xs

            else
                False

        [] ->
            True


init : Flags -> Url -> Nav.Key -> ( Model, Cmd Msg )
init flags url key =
    let
        loginFormURL =
            UrlBuilder.crossOrigin
                flags.idp
                [ "oauth2", "authorize" ]
                [ UrlBuilder.string "response_type" "code"
                , UrlBuilder.string "client_id" flags.authClientId
                , UrlBuilder.string "redirect_uri" flags.authRedirectURL
                ]

        endpoint =
            "/v1"

        initToken =
            if allJust [ flags.idToken, flags.accessToken, flags.refreshToken ] then
                Just
                    { idToken = Maybe.withDefault "" flags.idToken
                    , accessToken = Maybe.withDefault "" flags.accessToken
                    , refreshToken = Maybe.withDefault "" flags.refreshToken
                    }

            else
                Nothing

        authModel =
            Auth.init initToken flags.authClientId flags.idp flags.authRedirectURL
    in
    ( { key = key
      , route = Route.fromUrl url
      , windowSize = ( flags.windowWidth, flags.windowHeight )
      , loginFormURL = loginFormURL
      , logoutURL = UrlBuilder.crossOrigin flags.idp [ "logout" ] []
      , logoutRedirectURL = flags.logoutRedirectURL
      , endpoint = endpoint
      , authModel = authModel
      , filesModel = View.Files.init [] Nothing
      , fileUploadModel = View.FileUpload.init
      , headerModel = View.Org.Header.init loginFormURL Nothing
      , initUserModel = View.InitUser.init
      }
    , Cmd.batch
        [ Nav.pushUrl key (Url.toString url)
        , apiRequest endpoint authModel Api.GetUserRequest
        ]
    )


maybe : b -> (a -> b) -> Maybe a -> b
maybe default f =
    Maybe.withDefault default << Maybe.map f


maybeCmd : Maybe a -> (a -> Cmd msg) -> Cmd msg
maybeCmd ma f =
    maybe Cmd.none f ma


apiRequest : String -> Auth.Model -> Api.Request -> Cmd Msg
apiRequest endpoint authModel =
    Api.request ApiResponse endpoint authModel


apiResponse : Model -> Api.Request -> Result Http.Error Api.Response -> ( Model, Cmd Msg )
apiResponse model request result =
    case result of
        Ok response ->
            case response of
                Api.GetUserResponse res ->
                    ( model
                    , Task.perform HeaderMsg <| Task.succeed <| View.Org.Header.UpdateUser <| Just res.name
                    )

                Api.InitUserResponse ->
                    ( { model
                        | initUserModel = View.InitUser.init
                        , route = Route.Index
                      }
                    , Cmd.none
                    )

                Api.UploadResponse res ->
                    ( { model
                        | fileUploadModel = View.FileUpload.init
                        , route = Route.Files
                      }
                    , apiRequest model.endpoint model.authModel (Api.ListFilesRequest <| View.Files.continuationToken model.filesModel)
                    )

                Api.ListFilesResponse res ->
                    ( { model
                        | filesModel =
                            View.Files.init res.objects <|
                                if res.continuationToken == "" then
                                    Nothing

                                else
                                    Just res.continuationToken
                      }
                    , Cmd.none
                    )

                Api.GetFileResponse res ->
                    -- TODO show file detail
                    ( model, Cmd.none )

                Api.DeleteFileResponse res ->
                    ( model
                    , Task.perform ChangeRoute <| Task.succeed Route.Files
                    )

        Err (Http.BadStatus 401) ->
            ( model
            , Auth.tokenRequest RedirectToLoginForm
                (AuthMsg (RetryApiRequest request))
                model.authModel
                Nothing
            )

        Err (Http.BadStatus 404) ->
            case request of
                Api.GetUserRequest ->
                    -- user is not initialized
                    ( { model | route = Route.InitUserForm }
                    , Cmd.none
                    )

                _ ->
                    ( model, Cmd.none )

        Err err ->
            ( model, Cmd.none )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        UrlRequest urlRequest ->
            case urlRequest of
                Browser.Internal url ->
                    ( model, Nav.pushUrl model.key (Url.toString url) )

                Browser.External href ->
                    ( model, Nav.load href )

        UrlChanged url ->
            let
                route =
                    Route.fromUrl url
            in
            case route of
                Route.AuthCallback arg ->
                    ( model
                    , Auth.tokenRequest RedirectToLoginForm
                        (AuthMsg <| ChangeRoute Route.Index)
                        model.authModel
                        arg.code
                    )

                Route.Files ->
                    ( { model | route = route }
                    , apiRequest model.endpoint model.authModel (Api.ListFilesRequest Nothing)
                    )

                _ ->
                    ( { model | route = route }
                    , Cmd.none
                    )

        OnResize w h ->
            ( { model | windowSize = ( w, h ) }, Cmd.none )

        AuthResult prevMsg result ->
            case result of
                Ok token ->
                    ( model
                    , Cmd.batch
                        [ storeTokens ( token.idToken, token.accessToken, token.refreshToken )
                        , Task.perform identity <| Task.succeed prevMsg
                        ]
                    )

                Err err ->
                    case prevMsg of
                        RetryApiRequest req ->
                            case req of
                                Api.GetUserRequest ->
                                    ( { model | route = Route.Index }, Cmd.none )

                                _ ->
                                    ( model, Nav.load model.loginFormURL )

                        _ ->
                            ( model, Cmd.none )

        RedirectToLoginForm ->
            ( model, Nav.load model.loginFormURL )

        AuthMsg prevMsg authMsg ->
            let
                ( authModel, cmd ) =
                    Auth.update (AuthResult prevMsg) authMsg model.authModel
            in
            ( { model | authModel = authModel }
            , cmd
            )

        ChangeRoute route ->
            ( { model | route = route }
            , maybeCmd (Route.path route) <| Nav.pushUrl model.key
            )

        ApiResponse request result ->
            apiResponse model request result

        FileSelected key ->
            ( model
            , apiRequest model.endpoint model.authModel (Api.GetFileRequest key)
            )

        FileDelete key version ->
            ( model
            , apiRequest model.endpoint model.authModel (Api.DeleteFileRequest key { timestamp = version })
            )

        FileUploadRequested ->
            ( model
            , apiRequest model.endpoint
                model.authModel
                (Api.UploadRequest
                    { contentType = View.FileUpload.mime model.fileUploadModel
                    , blob = View.FileUpload.bytes model.fileUploadModel
                    }
                )
            )

        FileUploadMsg msg_ ->
            let
                ( fileUploadModel, cmd ) =
                    View.FileUpload.update FileUploadMsg msg_ model.fileUploadModel
            in
            ( { model | fileUploadModel = fileUploadModel }, cmd )

        HeaderMsg msg_ ->
            ( { model
                | headerModel = View.Org.Header.update msg_ model.headerModel
              }
            , Cmd.none
            )

        InitUserMsg msg_ ->
            ( { model | initUserModel = View.InitUser.update msg_ model.initUserModel }
            , Cmd.none
            )

        InitUserCommit req ->
            ( model
            , apiRequest model.endpoint model.authModel (Api.InitUserRequest req)
            )

        InitUserCancel ->
            ( { model
                | route = Route.Index
                , initUserModel = View.InitUser.init
              }
            , Cmd.none
            )

        RetryApiRequest req ->
            ( model, Api.request ApiResponse model.endpoint model.authModel req )

        NOP ->
            ( model, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Events.onResize OnResize


viewIndex _ =
    View.Index.view


viewFiles =
    View.Files.view
        { selected = FileSelected
        , delete = FileDelete
        , upload = ChangeRoute Route.FileUpload
        }


viewFileUpload =
    View.FileUpload.view FileUploadMsg FileUploadRequested


viewInitUserForm =
    View.InitUser.view
        { commit = InitUserCommit
        , cancel = InitUserCancel
        , toMsg = InitUserMsg
        }


template : Element msg -> Element msg -> Element msg
template header content =
    Element.column
        [ Element.centerX
        , Element.width Element.fill
        , Element.padding 5
        , Element.spacing 20
        ]
        [ header
        , content
        ]


view : Model -> Browser.Document Msg
view model =
    { title = "Uploader console"
    , body =
        [ Element.layout [] <|
            template
                (Lazy.lazy View.Org.Header.view model.headerModel)
                (case model.route of
                    Route.Index ->
                        Lazy.lazy viewIndex ()

                    Route.Files ->
                        Lazy.lazy viewFiles model.filesModel

                    Route.FileUpload ->
                        Lazy.lazy viewFileUpload model.fileUploadModel

                    Route.AuthCallback _ ->
                        Element.none

                    Route.InitUserForm ->
                        Lazy.lazy viewInitUserForm model.initUserModel

                    Route.NotFound url ->
                        Element.text "NotFound"
                )
        ]
    }


main : Program Flags Model Msg
main =
    Browser.application
        { init = init
        , update = update
        , subscriptions = subscriptions
        , view = view
        , onUrlChange = UrlChanged
        , onUrlRequest = UrlRequest
        }

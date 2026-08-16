module Api exposing
    ( Request(..)
    , Response(..)
    , request
    )

import Auth
import Http
import Json.Decode as Decode exposing (Decoder)
import Json.Encode as Encode
import Proto.Api
import Url.Builder as B


type Request
    = InitUserRequest Proto.Api.InitUserRequest
    | GetUserRequest
    | UploadRequest Proto.Api.UploadRequest
    | ListFilesRequest (Maybe String)
    | GetFileRequest String
    | DeleteFileRequest String Proto.Api.DeleteFileRequest


type Response
    = InitUserResponse
    | GetUserResponse Proto.Api.GetUserResponse
    | UploadResponse Proto.Api.UploadResponse
    | ListFilesResponse Proto.Api.ListResponse
    | GetFileResponse Proto.Api.GetFileResponse
    | DeleteFileResponse Proto.Api.DeleteFileResponse


type ResponseDecoder
    = Json (Decoder Response)
    | Empty Response


type alias Op =
    { method : String
    , path : String
    , body : Http.Body
    , decoder : ResponseDecoder
    }


maybe : b -> (a -> b) -> Maybe a -> b
maybe b f =
    Maybe.withDefault b << Maybe.map f


mkOp : Request -> Op
mkOp req =
    case req of
        InitUserRequest r ->
            { method = "POST"
            , path = B.absolute [ "user" ] []
            , body = Http.jsonBody <| Proto.Api.initUserRequestEncoder r
            , decoder = Empty InitUserResponse
            }

        GetUserRequest ->
            { method = "GET"
            , path = B.absolute [ "user" ] []
            , body = Http.emptyBody
            , decoder = Json <| Decode.map GetUserResponse Proto.Api.getUserResponseDecoder
            }

        UploadRequest r ->
            { method = "POST"
            , path = B.absolute [ "file" ] []
            , body = Http.jsonBody <| Proto.Api.uploadRequestEncoder r
            , decoder = Json <| Decode.map UploadResponse Proto.Api.uploadResponseDecoder
            }

        ListFilesRequest token ->
            { method = "GET"
            , path = B.absolute [ "file" ] <| maybe [] (List.singleton << B.string "token") token
            , body = Http.emptyBody
            , decoder = Json <| Decode.map ListFilesResponse Proto.Api.listResponseDecoder
            }

        GetFileRequest key ->
            { method = "GET"
            , path = B.absolute [ "file", key ] []
            , body = Http.emptyBody
            , decoder = Json <| Decode.map GetFileResponse Proto.Api.getFileResponseDecoder
            }

        DeleteFileRequest key r ->
            { method = "DELETE"
            , path = B.absolute [ "file", key ] []
            , body = Http.jsonBody <| Proto.Api.deleteFileRequestEncoder r
            , decoder = Json <| Decode.map DeleteFileResponse Proto.Api.deleteFileResponseDecoder
            }


request : (Request -> Result Http.Error Response -> msg) -> String -> Auth.Model -> Request -> Cmd msg
request toMsg endpoint authModel req =
    let
        op =
            mkOp req
    in
    Auth.signedRequest authModel
        { method = op.method
        , headers = []
        , url = endpoint ++ op.path
        , body = op.body
        , expect =
            case op.decoder of
                Json decoder ->
                    Http.expectJson (toMsg req) decoder

                Empty res ->
                    Http.expectString (toMsg req << Result.map (always res))
        }

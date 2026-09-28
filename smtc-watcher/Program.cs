using System;
using System.IO;
using System.Net.Http;
using System.Net.Http.Json;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Threading;
using System.Threading.Tasks;
using Windows.Media.Control;
using Meziantou.Framework.Win32;

public class Event
{
    [JsonPropertyName("title")]
    public string Title { get; set; }

    [JsonPropertyName("artist")]
    public string Artist { get; set; }

    [JsonPropertyName("album")]
    public string Album { get; set; }

    public Event(string title, string artist, string album)
    {
        Title = title;
        Artist = artist;
        Album = album;
    }
}

public class Credential
{
    [JsonPropertyName("token")]
    public string Token { get; set; } = string.Empty;

    [JsonPropertyName("user_id")]
    public string UserID { get; set; } = string.Empty;
}

class Program
{
    private const string TargetAppId = "Spotify";
    private const string ServerUrl = "http://172.19.164.243:5000";
    private const string CredentialAppName = "listening-engine-auth";

    private static GlobalSystemMediaTransportControlsSession? _currentSession;
    private static GlobalSystemMediaTransportControlsSessionManager? _sessionManager;

    private static string? _lastTitle;
    private static string? _lastArtist;
    private static string? _lastAlbumTitle;
    private static GlobalSystemMediaTransportControlsSessionPlaybackStatus? _lastStatus;

    private static readonly HttpClient client = new HttpClient();
    private static bool _isRegistered = false;
    private static string? userToken;
    private static string? userID;

    // Locks async tasks 1 by 1 so there is no overlap during property reads
    private static readonly SemaphoreSlim _asyncLock = new SemaphoreSlim(1, 1);

    static async Task Main()
    {
        try
        {
            Console.WriteLine("Finding user credentials...");
            _isRegistered = HaveCredentials();

            if (_isRegistered)
            {
                Console.WriteLine("Found! Setting up token...");
                SetToken();
                Console.WriteLine("Set!");
            }
            else
            {
                Console.WriteLine("Not found! Registering with server...");
                bool success = await RegisterCredentialsAsync();
                if (!success)
                {
                    Console.WriteLine("Failed to register with server. Exiting setup.");
                    return;
                }
            }

            Console.WriteLine("Initializing Windows Media Session Manager...");
            _sessionManager = await GlobalSystemMediaTransportControlsSessionManager.RequestAsync();
            Console.WriteLine("Session Manager Found!");

            _sessionManager.CurrentSessionChanged += OnCurrentSessionChanged;
            SyncActiveSession(_sessionManager.GetCurrentSession());

            Console.WriteLine("Listening engine actively tracking Spotify telemetry. Press Enter to exit.");
            Console.ReadLine();
        }
        catch (Exception ex)
        {
            Console.WriteLine($"Error receiving session manager: {ex.Message}");
        }
    }

    private static void OnCurrentSessionChanged(GlobalSystemMediaTransportControlsSessionManager sender, CurrentSessionChangedEventArgs args)
    {
        SyncActiveSession(sender.GetCurrentSession());
    }

    private static void SyncActiveSession(GlobalSystemMediaTransportControlsSession? session)
    {
        if (session == null || session == _currentSession)
        {
            return;
        }

        _currentSession = session;
        string appName = session.SourceAppUserModelId;

        if (appName.Contains(TargetAppId, StringComparison.OrdinalIgnoreCase))
        {
            Console.WriteLine("Spotify detected!");

            session.MediaPropertiesChanged -= OnMediaPropertiesChanged;
            session.MediaPropertiesChanged += OnMediaPropertiesChanged;

            session.PlaybackInfoChanged -= OnPlaybackStateChanged;
            session.PlaybackInfoChanged += OnPlaybackStateChanged;

            _ = GrabMediaDataAsync(session);
        }
    }

    private static bool HaveCredentials()
    {
        try
        {
            var cred = CredentialManager.ReadCredential(applicationName: CredentialAppName);
            return cred != null;
        }
        catch (Exception ex)
        {
            Console.WriteLine($"Unable to find credentials: {ex.Message}");
            return false;
        }
    }

    private static void SetToken()
    {
        try
        {
            var cred = CredentialManager.ReadCredential(applicationName: CredentialAppName);
            userToken = cred?.Password;
            userID = cred?.UserName;
        }
        catch (Exception ex)
        {
            Console.WriteLine($"Unable to find token: {ex.Message}");
        }
    }

    private static void OnMediaPropertiesChanged(GlobalSystemMediaTransportControlsSession sender, MediaPropertiesChangedEventArgs args)
    {
        Task.Run(async () =>
        {
            try
            {
                await GrabMediaDataAsync(sender);
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Unable to process media changes: {ex.Message}");
            }
        });
    }

    private static void OnPlaybackStateChanged(GlobalSystemMediaTransportControlsSession sender, PlaybackInfoChangedEventArgs args)
    {
        var playbackInfo = sender.GetPlaybackInfo().PlaybackStatus;
        if (_lastStatus != playbackInfo)
        {
            Console.WriteLine($"Playback status changed: {playbackInfo}");
        }

        _lastStatus = playbackInfo;
    }

    private static async Task<bool> RegisterCredentialsAsync()
    {
        try
        {
            using HttpRequestMessage tokenRequest = new HttpRequestMessage(HttpMethod.Post, ServerUrl);
            tokenRequest.Headers.Add("Auth-Token", "");
            tokenRequest.Headers.Add("User-ID", "");

            using var tokenResponse = await client.SendAsync(tokenRequest);
            tokenResponse.EnsureSuccessStatusCode();

            string responseBody = await tokenResponse.Content.ReadAsStringAsync();
            var options = new JsonSerializerOptions { PropertyNameCaseInsensitive = true };
            var credentials = JsonSerializer.Deserialize<Credential>(responseBody, options);

            if (credentials != null && !string.IsNullOrEmpty(credentials.Token))
            {
                Console.WriteLine("Creating new credentials for local machine...");
                CredentialManager.WriteCredential(
                    applicationName: CredentialAppName,
                    userName: credentials.UserID,
                    secret: credentials.Token,
                    comment: "Created credential for identity + auth for listening engine",
                    persistence: CredentialPersistence.LocalMachine
                );

                _isRegistered = HaveCredentials();
                SetToken();
                Console.WriteLine("Registration Complete!");
                return true;
            }

            throw new InvalidDataException("Unable to obtain valid credential payload from server.");
        }
        catch (Exception ex)
        {
            Console.WriteLine($"Registration Error: {ex.Message}");
            return false;
        }
    }

    private static async Task GrabMediaDataAsync(GlobalSystemMediaTransportControlsSession session)
    {
        Event? eventToSend = null;

        await _asyncLock.WaitAsync();
        try
        {
            if (!_isRegistered) return;

            var props = await session.TryGetMediaPropertiesAsync();

            if (props == null || string.IsNullOrWhiteSpace(props.Title) || string.IsNullOrWhiteSpace(props.Artist))
            {
                return;
            }

            if (props.Title == _lastTitle && props.Artist == _lastArtist && props.AlbumTitle == _lastAlbumTitle)
            {
                return;
            }

            _lastTitle = props.Title;
            _lastAlbumTitle = props.AlbumTitle;
            _lastArtist = props.Artist;

            eventToSend = new Event(props.Title, props.Artist, props.AlbumTitle);
        }
        finally
        {
            _asyncLock.Release();
        }

        if (eventToSend != null)
        {
            try
            {
                using HttpRequestMessage eventRequest = new HttpRequestMessage(HttpMethod.Post, ServerUrl)
                {
                    Content = JsonContent.Create(eventToSend)
                };

                eventRequest.Headers.Add("Auth-Token", userToken);
                eventRequest.Headers.Add("User-ID", userID);

                using HttpResponseMessage eventResponse = await client.SendAsync(eventRequest);
                if (eventResponse.IsSuccessStatusCode)
                {
                    Console.WriteLine($"[Sent Event] {eventToSend.Title} by {eventToSend.Artist}");
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"HTTP Error delivering telemetry: {ex.Message}");
            }
        }
    }
}
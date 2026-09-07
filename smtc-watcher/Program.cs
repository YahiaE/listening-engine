using System.Text.Json; 
using System.IO;
using Windows.Media.Control;
using System;
using System.Threading;
using System.Threading.Tasks;
using System.Net.Http;
using System.Net.Http.Json;
using System.Text;
using Meziantou.Framework.Win32;
using System.Security.Cryptography;

public class Event {
    public Guid? user_id {get; set;}
    public string song {get; set;}
    public string artist {get; set;}
    public string album {get; set;}
    public DateTime played_at {get; set;}

    public Event(string s, string at, string am, DateTime pa){
        song = s;
        artist = at;
        album = am;
        played_at = pa;
    }

   
}

 public class Credential
    {
        public string? UserID { get; set; }
        public string? Token { get; set; }
    }

class Program {

    private const string TargetAppId = "Spotify";
    private static GlobalSystemMediaTransportControlsSession? _currentSession;
    private static GlobalSystemMediaTransportControlsSessionManager? _sessionManager;
   
    private static string? _lastTitle;
    private static string? _lastArtist;
    private static string? _lastAlbumTitle;
    private static Windows.Media.Control.GlobalSystemMediaTransportControlsSessionPlaybackStatus? _lastStatus;

    private static readonly HttpClient client = new HttpClient();
    private static bool _isRegistered = false;
    private static string? userToken;
    // locks async 1 by 1 where there will be no overlap
    private static readonly SemaphoreSlim _asyncLock = new SemaphoreSlim(1, 1); 
    // private static Windows.Security.Credentials.PasswordCredential credentials;
    // private static Guid userID;
    static async Task Main(){
       
        try {
            Console.WriteLine("Initializing Windows Media Session Manager...");
            _sessionManager = await GlobalSystemMediaTransportControlsSessionManager.RequestAsync();
            Console.WriteLine("Session Manager Found!");

            Console.WriteLine("Finding user credentials...");
            _isRegistered = haveCredentials();
            setToken();

            /* 
            event += event handler
            when currentsessionchanged runs (when the session changes), 
            run the function oncurrentsessionchanged as well (the logic when the session changes)
            */
            _sessionManager.CurrentSessionChanged += OnCurrentSessionChanged;
        
            SyncActiveSession(_sessionManager.GetCurrentSession());

            Console.ReadLine();
        } catch (Exception ex){
            Console.WriteLine($"Error receiving session manager: {ex.Message}");
        }
    } 

    private static void OnCurrentSessionChanged(GlobalSystemMediaTransportControlsSessionManager sender, CurrentSessionChangedEventArgs args){
        SyncActiveSession(sender.GetCurrentSession());
    }

    private static void SyncActiveSession(GlobalSystemMediaTransportControlsSession? session){
        if (session == null){
            return;
        }

        if (session == _currentSession){
            return;
        }

        _currentSession = session;

        string appName = session.SourceAppUserModelId;

        if (appName.Contains(TargetAppId, StringComparison.OrdinalIgnoreCase)){
            Console.WriteLine("Spotify detected!");
            session.MediaPropertiesChanged -= OnMediaPropertiesChanged;
            session.MediaPropertiesChanged += OnMediaPropertiesChanged;

            session.PlaybackInfoChanged -= OnPlaybackStateChanged;
            session.PlaybackInfoChanged += OnPlaybackStateChanged;
            _ = GrabMediaDataAsync(session);
        }
    }

    private static bool haveCredentials(){
         try {
            var cred = CredentialManager.ReadCredential(applicationName: "listening-engine-auth");
            return true;
        } catch (Exception ex){
            Console.WriteLine($"Unable to find credentials: {ex.Message}");
            return false;
        }
    }

    private static void setToken(){
         try {
            var cred = CredentialManager.ReadCredential(applicationName: "listening-engine-auth");
            userToken = cred.Password;
        } catch (Exception ex){
            Console.WriteLine($"Unable to find token: {ex.Message}");
        }
    }

    private static async void OnMediaPropertiesChanged(GlobalSystemMediaTransportControlsSession sender, MediaPropertiesChangedEventArgs args){
        await GrabMediaDataAsync(sender);
    }

    private static void OnPlaybackStateChanged(GlobalSystemMediaTransportControlsSession sender, PlaybackInfoChangedEventArgs args){
        var playbackInfo = sender.GetPlaybackInfo().PlaybackStatus;
        if (_lastStatus != playbackInfo){
            Console.WriteLine(playbackInfo);
        }

        _lastStatus = playbackInfo;

        
    }

    private static async Task GrabMediaDataAsync(GlobalSystemMediaTransportControlsSession session){
        await _asyncLock.WaitAsync(); // unlock slot for async task

        try {
            
            var props = await session.TryGetMediaPropertiesAsync();

            if (props == null || props.Title == "" || (props.Title == _lastTitle && props.Artist == _lastArtist && props.AlbumTitle == _lastAlbumTitle)){
                return;
            }
        

            _lastTitle = props.Title;
            _lastAlbumTitle = props.AlbumTitle;
            _lastArtist = props.Artist;

            var newEvent = new Event(props.Title, props.Artist, props.AlbumTitle, DateTime.UtcNow);
            
            if (!_isRegistered){
                using HttpRequestMessage tokenRequest = new HttpRequestMessage(HttpMethod.Post,"http://172.19.164.243:5000");
                tokenRequest.Headers.Add("Auth-Token", "");
                using var tokenResponse = await client.SendAsync(tokenRequest);
                tokenResponse.EnsureSuccessStatusCode();

                string responseBody = await tokenResponse.Content.ReadAsStringAsync();
                Console.WriteLine(responseBody);
                
                var credentials = JsonSerializer.Deserialize<Credential>(responseBody);
                Console.WriteLine("Creating new credentials for local machine...");
                CredentialManager.WriteCredential(
                    applicationName: "listening-engine-auth",
                    userName: credentials.UserID,
                    secret: credentials.Token,
                    comment: "Created credential for identity + auth for listening engine",
                    persistence: CredentialPersistence.LocalMachine);
                Console.WriteLine("Complete! Checking if credentials exist in manager...");
                _isRegistered = haveCredentials();
                Console.WriteLine("Found!");
                setToken();
            } 
            
            using HttpRequestMessage eventRequest = new HttpRequestMessage(HttpMethod.Post,"http://172.19.164.243:5000"){
                Content = JsonContent.Create(newEvent)
            };

            
            eventRequest.Headers.Add("Auth-Token", userToken);
            using HttpResponseMessage eventResponse = await client.SendAsync(eventRequest);
            
        } catch (Exception ex){
            Console.WriteLine($"HTTP Error: {ex.Message}");
        } finally {
            _asyncLock.Release();
        }
    }
}


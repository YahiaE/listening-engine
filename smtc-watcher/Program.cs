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

public class Event {
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
        public string UserID { get; set; }
        public string Token { get; set; }
    }

class Program {

    private const string TargetAppId = "Spotify";
    private static GlobalSystemMediaTransportControlsSession? _currentSession;
    private static GlobalSystemMediaTransportControlsSessionManager? _sessionManager;
   
    private static string? _lastTitle;
    private static string? _lastArtist;
    private static string? _lastAlbumTitle;

    private static readonly HttpClient client = new HttpClient();
    private static bool isRegistered = false;
    // private static Windows.Security.Credentials.PasswordCredential credentials;
    // private static Guid userID;
    static async Task Main(){
       
        try {
            Console.WriteLine("Initializing Windows Media Session Manager...");
            _sessionManager = await GlobalSystemMediaTransportControlsSessionManager.RequestAsync();
            Console.WriteLine("Session Manager Found!");

            Console.WriteLine("Finding user credentials...");
            isRegistered = haveCredentials();

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
            Console.WriteLine($"ID: {cred.UserName}\nToken: {cred.Password}");
            return true;
        } catch (Exception ex){
            Console.WriteLine("Unable to find credentials");
            return false;
        }
    }

    private static string getToken(){
         try {
            var cred = CredentialManager.ReadCredential(applicationName: "listening-engine-auth");
            return cred.Password;
        } catch (Exception ex){
            Console.WriteLine("Unable to find token");
            return "";
        }
    }

    private static async void OnMediaPropertiesChanged(GlobalSystemMediaTransportControlsSession sender, MediaPropertiesChangedEventArgs args){
        await GrabMediaDataAsync(sender);
    }

    private static void OnPlaybackStateChanged(GlobalSystemMediaTransportControlsSession sender, PlaybackInfoChangedEventArgs args){
        var playbackInfo = sender.GetPlaybackInfo();
        Console.WriteLine(playbackInfo.PlaybackStatus);
    }

    private static async Task GrabMediaDataAsync(GlobalSystemMediaTransportControlsSession session){
        try {
            var props = await session.TryGetMediaPropertiesAsync();

            if (props == null || (props.Title == _lastTitle && props.Artist == _lastArtist && props.AlbumTitle == _lastAlbumTitle)){
                return;
            }
        

            _lastTitle = props.Title;
            _lastAlbumTitle = props.AlbumTitle;
            _lastArtist = props.Artist;

            var newEvent = new Event(props.Title, props.AlbumTitle, props.Artist, DateTime.UtcNow);
            

            
            

            if (!isRegistered){
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
                isRegistered = haveCredentials();
                Console.WriteLine("Found!");
            } else {
                Console.WriteLine("Already registered! Storing event...");
            }

            
            using HttpRequestMessage eventRequest = new HttpRequestMessage(HttpMethod.Post,"http://172.19.164.243:5000"){
                Content = JsonContent.Create(newEvent)
            };

            var userToken = getToken();
            eventRequest.Headers.Add("Auth-Token", userToken);


            using HttpResponseMessage eventResponse = await client.SendAsync(eventRequest);

            if (eventResponse.IsSuccessStatusCode)
            {
                Console.WriteLine("Data sent successfully!");
            }


            
           

            
            

            
            
            

           
            
        } catch (Exception ex){
            Console.WriteLine($"HTTP Error: {ex.Message}");
        }
    }
}


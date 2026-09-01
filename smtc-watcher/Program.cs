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
    public Guid? user_id {get; set;}
    public string song {get; set;}
    public string artist {get; set;}
    public string album {get; set;}
    public DateTime played_at {get; set;}

    public Event(Guid? ui, string s, string at, string am, DateTime pa){
        user_id = ui;
        song = s;
        artist = at;
        album = am;
        played_at = pa;
    }
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
    private static readonly Windows.Security.Credentials.PasswordVault vault = new Windows.Security.Credentials.PasswordVault();
    private static Windows.Security.Credentials.PasswordCredential credentials;
    private static Guid userID;
    static async Task Main(){
        try {
            credentials = vault.Retrieve(
                "listening-engine-auth-token",
                "user"
            );
            Console.WriteLine("Found existing user_id");
            credentials.RetrievePassword();

            var receivedID = credentials.Password;

            userID = new Guid(receivedID);
        } catch (Exception ex){
            Console.WriteLine("Unable to find auth token");
        }

        try {
            
            
            Console.WriteLine("Initializing Windows Media Session Manager...");
            _sessionManager = await GlobalSystemMediaTransportControlsSessionManager.RequestAsync();
            Console.WriteLine("Session Manager Found!");
             

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
        try {
            var props = await session.TryGetMediaPropertiesAsync();

            if (props == null || props.Title == "" || (props.Title == _lastTitle && props.Artist == _lastArtist && props.AlbumTitle == _lastAlbumTitle)){
                return;
            }
        

            _lastTitle = props.Title;
            _lastAlbumTitle = props.AlbumTitle;
            _lastArtist = props.Artist;

            Event newEvent = null;

            
            
            if (userID == null) {
                newEvent = new Event(null, props.Title, props.AlbumTitle, props.Artist, DateTime.UtcNow);  
            } else {
                newEvent = new Event(userID, props.Title, props.AlbumTitle, props.Artist, DateTime.UtcNow);
            }

            HttpResponseMessage response = await client.PostAsJsonAsync("http://172.19.164.243:5000", newEvent);
            response.EnsureSuccessStatusCode();

            byte[] responseBody = await response.Content.ReadAsByteArrayAsync();

            if (responseBody.Length > 0){
                Guid decodedID = new Guid(responseBody, bigEndian: true);
                credentials = new Windows.Security.Credentials.PasswordCredential(
                    resource: "listening-engine-auth-token",
                    userName: "user",
                    password: decodedID.ToString());
                vault.Add(credentials);
                userID = decodedID;
            }
            
            Console.WriteLine(newEvent.user_id);
            

        } catch (Exception ex){
            Console.WriteLine($"Error: {ex.Message}");
        }
    }
}


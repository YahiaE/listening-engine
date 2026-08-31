using System.Text.Json; 
using System.IO;
using Windows.Media.Control;
using System;
using System.Threading;
using System.Threading.Tasks;
using System.Net.Http;
using System.Net.Http.Json;

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

class Program {

    private const string TargetAppId = "Spotify";
    private static GlobalSystemMediaTransportControlsSession? _currentSession;
    private static GlobalSystemMediaTransportControlsSessionManager? _sessionManager;
   
    private static string? _lastTitle;
    private static string? _lastArtist;
    private static string? _lastAlbumTitle;

    private static readonly HttpClient client = new HttpClient();

    static async Task Main(){
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
            
            
            if (props.Title != "") {
                Console.WriteLine(props.Title + " seen");
                var newEvent = new Event(props.Title, props.AlbumTitle, props.Artist, DateTime.UtcNow);
                HttpResponseMessage response = await client.PostAsJsonAsync("http://172.19.164.243:5000", newEvent);
                response.EnsureSuccessStatusCode();

                string responseBody = await response.Content.ReadAsStringAsync();
                Console.WriteLine("Response received:");
                Console.WriteLine(responseBody);
            }
            

           
            
        } catch (Exception ex){
            Console.WriteLine($"Error: {ex.Message}");
        }
    }
}


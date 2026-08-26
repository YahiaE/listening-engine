using System.Text.Json; 
using Windows.Media.Control;
using System;
using System.Threading;
using System.Threading.Tasks;

class Program {

    private const string TargetAppId = "Spotify";
    private static GlobalSystemMediaTransportControlsSession? _currentSession;
    private static GlobalSystemMediaTransportControlsSessionManager _sessionManager;
    private static int _counter = 0;
    private static string? _lastSong;

    static async Task Main(){
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

            var song = $"{props.Title} by {props.Artist} from {props.AlbumTitle}";

            if (song == _lastSong){
                return;
            }

            _lastSong = song;
            _counter+=1;
            if (props != null){
                Console.WriteLine($"[{DateTime.Now:HH:mm:ss.fff}] " + $"Counter: {_counter} => {song}");
            }
        } catch (Exception ex){
            Console.WriteLine($"Error receiving properties: {ex.Message}");
        }
    }
}


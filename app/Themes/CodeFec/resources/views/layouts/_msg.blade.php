<link rel="stylesheet" href="{{ mix('plugins/Core/iziToast/css/iziToast.min.css') }}">
@foreach (['danger', 'warning', 'success', 'info'] as $msg)
    @if(session()->has($msg))
        <script src="{{ mix('plugins/Core/iziToast/js/iziToast.min.js') }}"></script>
        <script>
            if("{{$msg}}"==="success"){
                var c= "#63ed7a";
            }
            if("{{$msg}}"==="danger"){
                var c= "#dc3545";
            }
            if("{{$msg}}"==="warning"){
                var c= "#ffc107";
            }
            if("{{$msg}}"==="#17a2b8"){
                var c= "#ffc107";
            }
            if("{{$msg}}"==="info"){
                var c="#3abaf4";
            }
            iziToast.show({
                title: '{{$msg}}',
                message: '{{session()->get($msg)}}',
                color: c,
                position: 'topRight',
                messageColor : '#ffffff',
                titleColor : '#ffffff'
            });
        </script>
    @endif
@endforeach

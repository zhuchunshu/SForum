@foreach (['danger', 'warning', 'success', 'info','error'] as $msg)
    @if(session()->has($msg))
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
            if("{{$msg}}"==="error"){
                var c= "#dc3545";
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


@foreach(Itf()->get('admin-ui-msg') as $key => $value)
    @if(call_user_func($value['enable'])===true)
        <script>
            iziToast.{{$value['status']}}({
                title: '{{$value['status']}}',
                message: '{{call_user_func($value['msg'])}}',
                position:  "{{core_default(@$value['position'],'topRight')}}",
            });
        </script>
    @endif
@endforeach
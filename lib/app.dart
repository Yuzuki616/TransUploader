// This example demos the TextField/SelectableText widget and keyboard
// integration with the go-flutter text backend
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_easyloading/flutter_easyloading.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:path/path.dart' as path;

void main() => runApp(MyApp());

class MyApp extends StatelessWidget {
  // This widget is the root of your application.
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Cms ffmpeg',
      theme: ThemeData(
        // If the host is missing some fonts, it can cause the
        // text to not be rendered or worse the app might crash.
        fontFamily: 'NotoSansSC',
        primarySwatch: Colors.lightBlue,
        // useMaterial3: true
      ),
      home: MyStatefulWidget(),
      builder: EasyLoading.init(),
    );
  }
}

class MyStatefulWidget extends StatefulWidget {
  MyStatefulWidget({Key key}) : super(key: key);

  @override
  _MyStatefulWidgetState createState() => _MyStatefulWidgetState();
}

class _MyStatefulWidgetState extends State<MyStatefulWidget>
    with TickerProviderStateMixin {
  FocusNode myFocus = FocusNode();
  String selectFileText = "选择一个文件";
  String selectFilePath;
  String bgmId, episode;
  final bgmIdAndEpKey = GlobalKey<FormState>();

  @override
  void initState() {
    super.initState();
    loadSettings();
    listUploadTask();
  }

  Future<void> loadSettings() async {
    SharedPreferences prefs = await SharedPreferences.getInstance();
    var p = MethodChannel('yuzuki.io/ffmpeg', JSONMethodCodec());
    p.invokeMethod("setCmsSettings", {
      "baseUrl": prefs.getString("baseUrl"),
      "token": prefs.getString("token"),
      "typeId": prefs.getString("typeId"),
      "player": prefs.getString("player"),
    });
  }

  Future<void> cmsSettings() async {
    var pref;
    var baseUrl, token, player, typeId;
    try {
      pref = await SharedPreferences.getInstance();
      baseUrl = pref.getString("baseUrl");
      token = pref.getString("token");
      player = pref.getString("player");
      typeId = pref.getString("typeId");
    } catch (e) {
      print(e);
      return;
    }
    return showDialog<String>(
      context: context,
      builder: (BuildContext context) => AlertDialog(
        title: const Text('入库参数设置'),
        content: Wrap(
          children: [
            Padding(
              padding: EdgeInsets.all(4),
              child: SizedBox.fromSize(
                size: Size(200, 50),
                child: TextFormField(
                  initialValue: baseUrl,
                  decoration: InputDecoration(
                    border: OutlineInputBorder(),
                    labelText: 'BaseUrl',
                  ),
                  onChanged: (value) async {
                    baseUrl = value;
                  },
                ),
              ),
            ),
            Padding(
              padding: EdgeInsets.all(4),
              child: SizedBox.fromSize(
                size: Size(200, 50),
                child: TextFormField(
                  initialValue: token,
                  decoration: InputDecoration(
                    border: OutlineInputBorder(),
                    labelText: 'Token',
                  ),
                  onChanged: (value) async {
                    token = value;
                  },
                ),
              ),
            ),
            Padding(
              padding: EdgeInsets.all(4),
              child: SizedBox.fromSize(
                size: Size(200, 50),
                child: TextFormField(
                  initialValue: player,
                  decoration: InputDecoration(
                    border: OutlineInputBorder(),
                    labelText: 'Player',
                  ),
                  onChanged: (value) async {
                    player = value;
                  },
                ),
              ),
            ),
            Padding(
              padding: EdgeInsets.all(4),
              child: SizedBox.fromSize(
                size: Size(200, 50),
                child: TextFormField(
                  initialValue: typeId,
                  decoration: InputDecoration(
                    border: OutlineInputBorder(),
                    labelText: 'TypeId',
                  ),
                  onChanged: (value) async {
                    typeId = value;
                  },
                ),
              ),
            ),
          ],
        ),
        actions: <Widget>[
          TextButton(
            onPressed: () => Navigator.pop(context, 'Cancel'),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () async {
              EasyLoading.show(status: "保存中");
              try {
                pref.setString("baseUrl", baseUrl);
                pref.setString("token", token);
                pref.setString("player", player);
                pref.setString("typeId", typeId);
              } catch (e) {
                print(e);
                EasyLoading.showError("保存失败");
              } finally {
                await loadSettings();
                EasyLoading.showSuccess("保存成功");
              }
              Navigator.pop(context, 'OK');
            },
            child: const Text('OK'),
          ),
        ],
      ),
    );
  }

  Future<void> selectFile() async {
    var p = MethodChannel('plugins.flutter.io/image_picker');
    try {
      selectFilePath = await p.invokeMethod("pickVideo", {"source": 1});
    } catch (e) {
      print(e);
      return;
    }
    if (selectFilePath == null) {
      return;
    }
    setState(() {
      selectFileText = path.basename(selectFilePath);
    });
  }

  Future<void> addUploadTask(bool slice) async {
    if (selectFilePath == null) {
      EasyLoading.showError("请选择要上传的文件！");
      return;
    }
    if ((bgmId == null || episode == null) && !slice) {
      EasyLoading.showError("请填写番组Id和集数！");
      return;
    }
    EasyLoading.show(status: "添加中...");
    var p = MethodChannel('yuzuki.io/ffmpeg', JSONMethodCodec());
    var pref = await SharedPreferences.getInstance();
    var payload = {
      "baseUrl": pref.getString("baseUrl"),
      "token": pref.getString("token"),
      "bangumiId": bgmId,
      "episode": episode,
      "path": selectFilePath
    };
    if (slice) {
      payload["isSlice"] = "1";
    }
    try {
      await p.invokeMethod("addUploadTask", payload);
    } catch (e) {
      print(e);
      EasyLoading.showError(e.toString());
      return;
    }
    EasyLoading.showSuccess("已添加至上传队列");
  }

  var tasks = [];

  Future<void> listUploadTask() async {
    var p = MethodChannel('yuzuki.io/ffmpeg', JSONMethodCodec());
    var tempTasks;
    while (true) {
      await Future.delayed(Duration(seconds: 3));
      try {
        tempTasks = await p.invokeMethod("listUploadTask");
      } catch (e) {
        print(e);
        continue;
      }
      if (listEquals(tempTasks, tasks)) {
        continue;
      }
      setState(() {
        tasks = tempTasks;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        width: double.infinity,
        child: Padding(
          padding: EdgeInsets.all(16),
          child: Row(children: [
            Expanded(
              flex: 2,
              child: Column(children: [
                Card(
                  margin: EdgeInsets.fromLTRB(10, 0, 10, 0),
                  child: Padding(
                    padding: EdgeInsets.all(12),
                    child: Column(children: [
                      Title(color: Colors.black, child: Text('选择文件')),
                      const Divider(),
                      ElevatedButton(
                          onPressed: () async => {selectFile()},
                          child: Text(selectFileText)),
                    ]),
                  ),
                ),
                Card(
                  margin: EdgeInsets.fromLTRB(10, 10, 10, 0),
                  child: Padding(
                    padding: EdgeInsets.all(12),
                    child: Column(children: [
                      Title(color: Colors.black, child: Text('上传参数设置')),
                      const Divider(),
                      Wrap(children: [
                        Padding(
                          padding: EdgeInsets.all(4),
                          child: ElevatedButton(
                            onPressed: cmsSettings,
                            child: Text("打开CMS入库设置"),
                          ),
                        ),
                        Padding(
                          padding: EdgeInsets.all(4),
                          child: ElevatedButton(
                            child: Text("打开上传接口设置"),
                            onPressed: () {
                              EasyLoading.showError('限制使用');
                            },
                          ),
                        ),
                      ]),
                      Form(
                        key: bgmIdAndEpKey,
                        child: Wrap(children: [
                          Padding(
                            padding: EdgeInsets.all(4),
                            child: SizedBox.fromSize(
                              size: Size(130, 40),
                              child: TextFormField(
                                decoration: InputDecoration(
                                  border: OutlineInputBorder(),
                                  labelText: '请输入番组ID',
                                ),
                                onChanged: (value) async {
                                  bgmId = value;
                                },
                              ),
                            ),
                          ),
                          Padding(
                            padding: EdgeInsets.all(4),
                            child: SizedBox.fromSize(
                              size: Size(130, 40),
                              child: TextFormField(
                                decoration: InputDecoration(
                                  border: OutlineInputBorder(),
                                  labelText: '请输入集数',
                                ),
                                onChanged: (value) async {
                                  episode = value;
                                },
                              ),
                            ),
                          ),
                        ]),
                      ),
                    ]),
                  ),
                ),
                Card(
                  margin: EdgeInsets.fromLTRB(10, 10, 10, 0),
                  child: Padding(
                    padding: EdgeInsets.all(12),
                    child: Column(children: [
                      Title(color: Colors.black, child: Text('操作')),
                      const Divider(),
                      Wrap(children: [
                        Padding(
                          padding: EdgeInsets.all(4),
                          child: ElevatedButton(
                              onPressed: () => addUploadTask(false),
                              child: Text("单文件上传")),
                        ),
                        Padding(
                          padding: EdgeInsets.all(4),
                          child: ElevatedButton(
                              onPressed: () => addUploadTask(true),
                              child: Text("切片上传")),
                        ),
                      ]),
                    ]),
                  ),
                ),
              ]),
            ),
            Expanded(
              flex: 1,
              child: Padding(
                padding: EdgeInsets.fromLTRB(8, 0, 8, 0),
                child: Card(
                  child: Padding(
                    padding: EdgeInsets.all(12),
                    child: Column(children: [
                      Title(color: Colors.black, child: Text('上传任务列表')),
                      const Divider(),
                      Expanded(
                        child: ListView.builder(
                          itemBuilder: (context, index) {
                            if (tasks == null) {
                              return null;
                            }
                            if (tasks.length <= index) {
                              return null;
                            }
                            return ListTile(
                              title: Text(
                                tasks[index]["name"],
                                overflow: TextOverflow.ellipsis,
                              ),
                              subtitle: Text(
                                tasks[index]["status"],
                              ),
                              /*trailing: IconButton(
                                icon: Icon(Icons.delete),
                              ),*/
                            );
                          },
                        ),
                      ),
                    ]),
                  ),
                ),
              ),
            ),
          ]),
        ),
      ),
    );
  }
}

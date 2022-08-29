package jsc

import (
	"testing"
)

func Test_Must_Return_Component_Lifecycle(t *testing.T) {

	template := `
    <div>
      <div></div>
      <script>
          const variable = () => {}
    
          const OnMount = () => { }  
          function BeforeUpdate(){ }
          let AfterUpdate = new SomeConstructor(); 
          var BeforeRender = window.SomeReference;
          const AfterRender = variable
          const [OnDestroy, OnConnect] =  [() => { }, () => { }]  
          let OnDisconnect = param.init()
          let OnError, OnEvent;      
      </script>
    </div>
  `

	expected := `
    STX.s('#fWVwntaPY84', function (STX) {
      const _$line = 3;
      const _$file = "template.html";
    
      return {
        f: _$file,
        l: _$line,
        i : function ($) {
    
          // Component
          const variable = () => { }; 
          const OnMount = () => { }; 
          function BeforeUpdate () { }; 
          let AfterUpdate = new SomeConstructor(); 
          var BeforeRender = window.SomeReference; 
          const AfterRender = variable; 
          const [OnDestroy,OnConnect] = [() => { }, () => { }]; 
          let OnDisconnect = param.init(); 
          let OnError, OnEvent; 
    
          return {
            a : OnMount,
            b : BeforeUpdate,
            c : AfterUpdate,
            d : BeforeRender,
            e : AfterRender,
            f : OnDestroy,
            g : OnConnect,
            h : OnDisconnect,
            i : OnError,
            j : OnEvent
          };
        }
      }
    })
  `

	testCompileJs(t, template, expected, nil)
}

func Test_Exports(t *testing.T) {

	template := `
    <div>
      <div></div>
      <script>
          const variable = () => {};
    
          export const MyVariableA = () => { };
          export function MyVariableB(){ };
          export let MyVariableC = new SomeConstructor();
          export var MyVariableD = window.SomeReference;
          export const MyVariableE = variable;
          const [MyVariableF, MyVariableG] =  [() => { }, () => { }];
          let MyVariableH = param.init();
          let MyVariableI, MyVariableJ;
          export {
            MyVariableF as MyVariableRenamed,
            MyVariableG,
            MyVariableH,
            MyVariableI
          }
      </script>
    </div>
  `

	expected := `
    STX.s('#OhAUjOH4ihc', function (STX) {
      const _$line = 3;
      const _$file = "template.html";
    
      return {
        f: _$file,
        l: _$line,
        i : function ($) {
    
          // Component
          const variable = () => { }; 
          const MyVariableA = () => { }
          function MyVariableB(){ }
          let MyVariableC = new SomeConstructor();
          var MyVariableD = window.SomeReference;
          const MyVariableE = variable
          const [MyVariableF, MyVariableG] =  [() => { }, () => { }]
          let MyVariableH = param.init()
          let MyVariableI, MyVariableJ;
          return {
            z : {
              MyVariableA : MyVariableA,
              MyVariableB : MyVariableB,
              MyVariableC : MyVariableC,
              MyVariableD : MyVariableD,
              MyVariableE : MyVariableE,
              MyVariableRenamed : MyVariableF,
              MyVariableG : MyVariableG,
              MyVariableH : MyVariableH,
              MyVariableI : MyVariableI
            }
          };
        }
      }
    })
  `

	testCompileJs(t, template, expected, nil)
}

func Test_Export_Default_Function(t *testing.T) {

	template := `
    <div>
      <div></div>
      <script>
          const variable = () => {};
          export default function(){ return { variable: variable} }
      </script>
    </div>
  `

	expected := `
    STX.s('#hzi-8ltxy2w', function (STX) {
      const _$line = 3;
      const _$file = "template.html";
    
      return {
        f: _$file,
        l: _$line,
        i : function ($) {
          const variable = () => { }; 
          return {
            z : function(){ return { variable: variable} }
          };
        }
      }
    })
  `

	testCompileJs(t, template, expected, nil)
}

func Test_Export_Default_Object(t *testing.T) {

	template := `
    <div>
      <div></div>
      <script>
          const variable = () => {};
          export const ignored = 33;
          export default {
            MyVariableRenamed: variable,
            variable
          }
          export const ignored2 = 33;
      </script>
    </div>
  `

	expected := `
    STX.s('#_47GUTOYPc1I', function (STX) {
      const _$line = 3;
      const _$file = "template.html";
    
      return {
        f: _$file,
        l: _$line,
        i : function ($) {
    
          // Component
          const variable = () => { }; 
          const ignored = 33;
          const ignored2 = 33;
          return {
            z : {
              MyVariableRenamed: variable,
              variable
            }
          };
        }
      }
    })
  `

	testCompileJs(t, template, expected, nil)
}

func Test_Export_Default_Variable(t *testing.T) {

	template := `
    <div>
      <div></div>
      <script>
          const variable = () => {};
          export const ignored = 33;
          export default variable
          export const ignored2 = 33;
      </script>
    </div>
  `

	expected := `
    STX.s('#viavweubtio', function (STX) {
      const _$line = 3;
      const _$file = "template.html";
    
      return {
        f: _$file,
        l: _$line,
        i : function ($) {
    
          // Component
          const variable = () => { }; 
          const ignored = 33;
          const ignored2 = 33;
          return {
            z : variable
          };
        }
      }
    })
  `

	testCompileJs(t, template, expected, nil)
}

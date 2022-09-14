package jsc

import (
	"testing"
)

func Test_must_return_component_lifecycle(t *testing.T) {

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
        c : function ($, STX) {
    
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
            j : OnError,
            i : OnEvent
          };
        }
      }
    })
  `

	testCompileJs(t, template, expected, nil)
}

func Test_exports(t *testing.T) {

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
        c : function ($, STX) {
    
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

func Test_export_default_function(t *testing.T) {

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
        c : function ($, STX) {
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

func Test_export_default_object(t *testing.T) {

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
        c : function ($, STX) {
    
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

func Test_export_default_variable(t *testing.T) {

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
        c : function ($, STX) {
    
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

func Test_assignments(t *testing.T) {

	template := `
    <div>
      <div></div>
      <script>
        let value = 1;

        // Unary Assignment
        const fn1 = () => { value++ }
        const fn2 = () => { value-- }
        const fn3 = () => { ++value }
        const fn4 = () => { --value }
        const fn5 = () => { --value + value++ }

        // Binary Assignment
        const fnA = () => { value    = 25; } // =    Assignment
        const fnB = () => { value   += 25; } // +=   Addition assignment
        const fnC = () => { value   *= 25; } // *=   Multiplication assignment
        const fnD = () => { value   /= 25; } // /=   Division assignment
        const fnE = () => { value   -= 25; } // -=   Subtraction assignment
        const fnF = () => { value   %= 25; } // %=   Remainder assignment
        const fnG = () => { value  **= 25; } // **=  Exponentiation assignment (ECMAScript 2016)
        const fnH = () => { value  <<= 25; } // <<=  Left shift assignment
        const fnI = () => { value  >>= 25; } // >>=  Right shift assignment
        const fnJ = () => { value >>>= 25; } // >>>= Unsigned right shift assignment
        const fnK = () => { value   &= 25; } // &=   Bitwise AND assignment
        const fnL = () => { value   ^= 25; } // ^=   Bitwise XOR assignment
        const fnM = () => { value   |= 25; } // |=   Bitwise OR assignment
        const fnN = () => { value  &&= 25; } // &&=  Logical AND assignment
        const fnO = () => { value  ||= 25; } // ||=  Logical OR assignment
        const fnP = () => { value  ??= 25; } // ??=  Logical nullish assignment

        // call
        myFunctionCall(value++)
        myFunctionCall(++value)
        value = myFunctionCall(value++)

        const fnx = () => {
          myFunctionCall(value++)
          myFunctionCall(++value)
          value = myFunctionCall(value++)
        }
      </script>
    </div>
  `

	expected := `
    STX.s('#BQZjdgQ1bZI', function (STX) {
      const _$line = 3;
      const _$file = "template.html";
      return {
        f: _$file, 
        l: _$line, 
        c: function ($, STX) {
          let value = 1;

          // Unary Assignment
          const fn1 = () => { $.i(0, value, (value++, value)); };
          const fn2 = () => { $.i(0, value, (value--, value)); };
          const fn3 = () => { $.i(0, value, ++value); };
          const fn4 = () => { $.i(0, value, --value); };
          const fn5 = () => { $.i(0, value, --value) + $.i(0, value, (value++, value)); };

          // Binary Assignment
          const fnA = () => {    $.i(0, value, value = 25); }; // =    Assignment
          const fnB = () => {   $.i(0, value, value += 25); }; // +=   Addition assignment
          const fnC = () => {   $.i(0, value, value *= 25); }; // *=   Multiplication assignment
          const fnD = () => {   $.i(0, value, value /= 25); }; // /=   Division assignment
          const fnE = () => {   $.i(0, value, value -= 25); }; // -=   Subtraction assignment
          const fnF = () => {   $.i(0, value, value %= 25); }; // %=   Remainder assignment
          const fnG = () => {  $.i(0, value, value **= 25); }; // **=  Exponentiation assignment (ECMAScript 2016)
          const fnH = () => {  $.i(0, value, value <<= 25); }; // <<=  Left shift assignment
          const fnI = () => {  $.i(0, value, value >>= 25); }; // >>=  Right shift assignment
          const fnJ = () => { $.i(0, value, value >>>= 25); }; // >>>= Unsigned right shift assignment
          const fnK = () => {   $.i(0, value, value &= 25); }; // &=   Bitwise AND assignment
          const fnL = () => {   $.i(0, value, value ^= 25); }; // ^=   Bitwise XOR assignment
          const fnM = () => {   $.i(0, value, value |= 25); }; // |=   Bitwise OR assignment
          const fnN = () => {  $.i(0, value, value &&= 25); }; // &&=  Logical AND assignment
          const fnO = () => {  $.i(0, value, value ||= 25); }; // ||=  Logical OR assignment
          const fnP = () => {  $.i(0, value, value ??= 25); }; // ??=  Logical nullish assignment

          // call
          myFunctionCall($.i(0, value, (value++, value)));
          myFunctionCall($.i(0, value, ++value));
          $.i(0, value, value = myFunctionCall($.i(0, value, (value++, value))));

          const fnx = () => {
            myFunctionCall($.i(0, value, (value++, value)));
            myFunctionCall($.i(0, value, ++value));
            $.i(0, value, value = myFunctionCall($.i(0, value, (value++, value))));
          };
          return {};
        }
      };
    });
  `

	testCompileJs(t, template, expected, nil)
}

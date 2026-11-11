import { TextStyle } from '@tiptap/extension-text-style';

/**
 * TextStyle do texto do jogo: além do `color` CSS (usado pelo Tiptap para
 * pintar), guarda a TAG canônica da cor no atributo `gameColor`. No FF13 ela
 * carrega os bytes do idioma (ex.: `{CLR:F9:40:White}`); no FFX, `{CLR:BLUE}`.
 *
 * Sem isso, o serializer só teria o hex e duas cores com o mesmo hex
 * (Ex00..28, White/PureWhite) perderiam o byte exato.
 */
export const GameTextStyle = TextStyle.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      gameColor: {
        default: null,
        parseHTML: (element) => element.getAttribute('data-game-color'),
        renderHTML: (attributes) => {
          if (!attributes.gameColor) return {};
          return { 'data-game-color': attributes.gameColor as string };
        },
      },
    };
  },
});
